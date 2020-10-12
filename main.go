package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"time"

	"github.com/Netflix/go-env"
	"github.com/diamondburned/ffsync/ffmpeg"
	"github.com/diamondburned/ffsync/ffmpeg/cover"
	"github.com/diamondburned/ffsync/ffmpeg/opus"
	"github.com/diamondburned/ffsync/internal/telemetry"
	"github.com/diamondburned/ffsync/internal/telemetry/fallback"
	"github.com/diamondburned/ffsync/internal/telemetry/influx"
	"github.com/diamondburned/ffsync/sync"
	"github.com/pkg/errors"
)

func main() {
	var config struct {
		Influx influx.Config

		Formats     string `env:"FFSYNC_FORMATS"`
		CopyFormats string `env:"FFSYNC_COPY_FORMATS"`
		Frequency   string `env:"FFSYNC_FREQUENCY"`
	}

	_, err := env.UnmarshalFromEnviron(&config)
	if err != nil {
		log.Fatalln("Failed to load env:", err)
	}

	var cfg = sync.Options{
		FileFormats: []string{".mp3", ".flac", ".aac", ".ogg", ".opus"},
		CopyFormats: []string{".jpg", ".jpeg", ".png"},
	}

	if config.Formats != "" {
		cfg.FileFormats = strings.Split(config.Formats, ",")
	}
	if config.CopyFormats != "" {
		cfg.CopyFormats = strings.Split(config.CopyFormats, ",")
	}

	var wfreq = time.Minute
	if config.Frequency != "" {
		f, err := time.ParseDuration(config.Frequency)
		if err != nil {
			log.Fatalln("Failed to parse frequency:", err)
		}
		wfreq = f
	}

	if len(os.Args) < 3 {
		log.Fatalln("Invalid usage. Usage:", filepath.Base(os.Args[0]), "src dst")
	}

	var t telemetry.Telemeter
	if config.Influx.Address != "" {
		t, err = influx.NewClient(config.Influx)
		if err != nil {
			log.Fatalln("InfluxDB error:", err)
		}
	} else {
		t = fallback.New()
	}

	defer t.Close()

	a := &Application{
		Telemeter: t,
	}

	s, err := sync.New(os.Args[1], os.Args[2], cfg, a)
	if err != nil {
		log.Fatalln("Failed to make a new syncer:", err)
	}

	if err := s.Start(wfreq); err != nil {
		log.Fatalln("Failed to run syncer:", err)
	}
	defer s.Close()

	sig := make(chan os.Signal)
	signal.Notify(sig, os.Interrupt)
	<-sig
}

type Application struct {
	Telemeter telemetry.Telemeter
}

func (a *Application) ConvertExt(name string) string {
	return ffmpeg.ConvertExt(name, "opus")
}

func (a *Application) ConvertCtx(ctx context.Context, src, dst string) (err error) {
	// Only derive the album art if the cover does not exist.
	if _, exists := cover.ExistsAlbum(dst); !exists {
		var done = make(chan error, 1)

		// On function exit, wait for the ExtractAlbum goroutine to finish.
		defer func() {
			// Only use cover's error if opus' error is nil.
			if coverErr := <-done; err != nil {
				err = coverErr
			}
		}()

		// Extract the album art in another goroutine.
		go func() {
			var coverErr error
			defer func() { done <- coverErr }()

			coverSubmitter := a.submitter(src, "cover")

			o, err := cover.ExtractAlbum(ctx, src, dst)
			if err != nil {
				if !cover.ErrIsNoStream(err) {
					coverErr = errors.Wrap(err, "failed to extract album art")
				}
				return
			}

			coverSubmitter(o)
		}()
	}

	opusSubmitter := a.submitter(src, "opus")

	o, err := opus.ConvertCtx(ctx, src, dst)
	if err != nil {
		err = errors.Wrap(err, "failed to convert")
		return
	}

	opusSubmitter(o)

	return nil
}

func (a *Application) submitter(src, rType string) func(*ffmpeg.Result) {
	var now = time.Now()

	return func(result *ffmpeg.Result) {
		var dura = time.Now().Sub(now)

		if result.Progress.Progress != ffmpeg.ProgressEnd {
			return
		}

		a.Telemeter.WriteDuration(dura, "convert", telemetry.Extras{
			"encoded":       result.OutDuration().Milliseconds(),
			"runtime":       result.Runtime.Milliseconds(),
			"realtime_mult": result.Speed,
			"wrote_bytes":   result.TotalSize,
			"bitrate":       result.Bitrate,
			"src":           src,
			"dst":           result.OutputPath,
		})
	}
}
