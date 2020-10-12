package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/Netflix/go-env"
	"github.com/diamondburned/ffsync/ffmpeg"
	"github.com/diamondburned/ffsync/ffmpeg/cover"
	"github.com/diamondburned/ffsync/ffmpeg/opus"
	"github.com/diamondburned/ffsync/internal/osutil"
	"github.com/diamondburned/ffsync/internal/telemetry"
	"github.com/diamondburned/ffsync/internal/telemetry/fallback"
	"github.com/diamondburned/ffsync/internal/telemetry/influx"
	"github.com/diamondburned/ffsync/sync"
	"golang.org/x/sync/semaphore"
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
		ErrorLog: func(err error) {
			log.Println("[sync]", err)
		},
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
		Telemeter:       t,
		CopySemaphore:   *semaphore.NewWeighted(64),
		FFmpegSemaphore: *semaphore.NewWeighted(int64(runtime.GOMAXPROCS(-1))),
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
	Telemeter       telemetry.Telemeter
	CopySemaphore   semaphore.Weighted
	FFmpegSemaphore semaphore.Weighted
}

func (a *Application) ConvertExt(name string) string {
	return ffmpeg.ConvertExt(name, "opus")
}

func (a *Application) QueueCopy(src, dst string) {
	semaJob(time.Minute, &a.CopySemaphore, func(ctx context.Context) {
		if err := osutil.Copy(ctx, src, dst); err != nil {
			log.Println("[copy]", err)
		}
	})
}

func (a *Application) QueueConvert(src, dst string) {
	// Only derive the album art if the cover does not exist.
	if _, exists := cover.ExistsAlbum(dst); !exists {
		semaJob(time.Minute, &a.FFmpegSemaphore, func(ctx context.Context) {
			coverSubmitter := a.submitter(src, "cover")

			o, err := cover.ExtractAlbum(ctx, src, dst)
			if err != nil {
				if !cover.ErrIsNoStream(err) {
					log.Println("[cover] failed to extract album art:", err)
				}
				return
			}

			coverSubmitter(o)
		})
	}

	semaJob(10*time.Minute, &a.FFmpegSemaphore, func(ctx context.Context) {
		opusSubmitter := a.submitter(src, "opus")

		o, err := opus.ConvertCtx(ctx, src, dst)
		if err != nil {
			log.Println("[opus] failed to convert:", err)
			return
		}

		opusSubmitter(o)
	})
}

func (a *Application) submitter(src, rType string) func(*ffmpeg.Result) {
	var now = time.Now()

	return func(result *ffmpeg.Result) {
		var dura = time.Now().Sub(now)

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

// semaJob blocks until a semaphore is acquired, then runs fn in a goroutine.
func semaJob(t time.Duration, sema *semaphore.Weighted, fn func(context.Context)) {
	// 1 minute timeout.
	ctx, cancel := context.WithTimeout(context.TODO(), time.Minute)

	if err := sema.Acquire(ctx, 1); err != nil {
		log.Println("[copy] failed to acquire sema")
		cancel()
		return
	}

	go func() {
		defer cancel()
		defer sema.Release(1)

		fn(ctx)
	}()
}
