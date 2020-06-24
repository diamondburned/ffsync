package main

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Netflix/go-env"
	"github.com/diamondburned/ffsync/opus"
	"github.com/diamondburned/ffsync/opus/parse"
	"github.com/diamondburned/ffsync/sync"
	"github.com/diamondburned/ffsync/telemetry"
	"github.com/diamondburned/ffsync/telemetry/fallback"
	"github.com/diamondburned/ffsync/telemetry/influx"
	"github.com/pkg/errors"
)

func main() {
	var config struct {
		Influx influx.Config

		Formats   string `env:"FFSYNC_FORMATS"`
		Frequency string `env:"FFSYNC_FREQUENCY"`
	}

	_, err := env.UnmarshalFromEnviron(&config)
	if err != nil {
		log.Fatalln("Failed to load env:", err)
	}

	var fmts = []string{".mp3", ".flac", ".aac"}
	if config.Formats != "" {
		fmts = strings.Split(config.Formats, ",")
	}

	var wfreq = 2 * time.Second
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
		OpusPool:  opus.NewPool(),
	}

	s, err := sync.New(os.Args[1], os.Args[2], fmts, a)
	if err != nil {
		log.Fatalln("Failed to make a new syncer:", err)
	}
	s.Error = t.Error

	if err := s.Run(wfreq); err != nil {
		log.Fatalln("Failed to run syncer:", err)
	}
}

type Application struct {
	Telemeter telemetry.Telemeter
	OpusPool  *opus.Pool
}

func (a *Application) ConvertExt(name string) string {
	return opus.ConvertExt(name, "opus")
}

func (a *Application) ConvertCtx(ctx context.Context, src, dst string) error {
	var now = time.Now()

	o, err := a.OpusPool.ConvertCtx(ctx, src, dst)
	if err != nil {
		return errors.Wrap(err, "Failed to convert")
	}

	var dur = time.Now().Sub(now)

	enc, err := parse.Parse(o)
	if err != nil {
		return errors.Wrap(err, "Failed to parse opusenc")
	}

	a.Telemeter.WriteDuration(dur, "convert", telemetry.Extras{
		"encoded":       enc.Encoded.Milliseconds(),
		"runtime":       enc.Runtime.Milliseconds(),
		"realtime_mult": enc.RealtimeMult,
		"wrote_bytes":   enc.WroteBytes,
		"bitrate":       enc.Bitrate,
		"overhead":      enc.Overhead,
	})

	return nil
}
