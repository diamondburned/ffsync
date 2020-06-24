package opus

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"golang.org/x/sync/semaphore"
)

func init() {
	for _, arg0 := range []string{"ffmpeg", "opusenc"} {
		if _, err := exec.LookPath(arg0); err != nil {
			log.Fatalln("Failed to find", arg0+".")
		}
	}
}

const (
	CoverArtSz = "600"
	CoverArtQ  = "3"
)

var (
	vf = fmt.Sprintf("scale=-1:'min(%s,ih)'", CoverArtSz)
)

type Pool struct {
	sema semaphore.Weighted
}

func NewPool() *Pool {
	return &Pool{
		sema: *semaphore.NewWeighted(int64(runtime.GOMAXPROCS(-1))),
	}
}

func (p *Pool) Convert(src, dst string) (string, error) {
	return p.ConvertCtx(context.Background(), src, dst)
}

func (p *Pool) ConvertCtx(ctx context.Context, src, dst string) (string, error) {
	if err := p.sema.Acquire(ctx, 1); err != nil {
		return "", err
	}
	defer p.sema.Release(1)

	return ConvertCtx(ctx, src, dst)
}

func Convert(src, dst string) (string, error) {
	return ConvertCtx(context.Background(), src, dst)
}

// ConvertCtx atomically converts src to dst as an opus file.
func ConvertCtx(ctx context.Context, src, dst string) (string, error) {
	// The path to a temporary file, which is basically the same path but with a
	// dot prepended to the filename: /path/to/.file
	tmpdst := filepath.Join(filepath.Dir(dst), "."+filepath.Base(dst))

	// Convert and write to that temp file.
	o, err := convertCtx(ctx, src, tmpdst)
	if err != nil {
		os.Remove(tmpdst)
		return o, err
	}

	// Atomically rename that temp file to the intended destination.
	return o, os.Rename(tmpdst, dst)
}

func convertCtx(ctx context.Context, src, dst string) (string, error) {
	ffmpeg := exec.CommandContext(ctx,
		"ffmpeg", "-loglevel", "warning", "-hide_banner", "-threads", "1",
		// Input file
		"-i", src,
		// Output format and options
		"-f", "flac", "-sample_fmt", "s16",
		// Album art options
		"-c:v", "mjpeg", "-sws_flags", "lanczos", "-q:v", CoverArtQ, "-vf", vf, "-vsync", "0",
		// Audio encoding options (FLAC to passthrough to opusenc)
		"-c:a", "flac", "-compression_level", "0",
		// stdout
		"-",
	)
	// Use a custom local stderr buffer.
	ffmpegErr := FFmpegError{}
	ffmpeg.Stderr = ffmpegErr.Stderr()

	opusenc := exec.CommandContext(ctx,
		"opusenc", "--bitrate", "96", "--music", "--downmix-stereo", "-", dst,
	)
	// Errors only happen when we run ffmpeg, so we don't need to check that
	// here.
	opusenc.Stdin, _ = ffmpeg.StdoutPipe()
	// Use a custom local stderr buffer.
	opusencErr := OpusencError{}
	opusenc.Stderr = opusencErr.Stderr()

	// Start the listener first.
	if err := opusenc.Start(); err != nil {
		return "", opusencErr.Wrap(err)
	}

	// Run the ffmpeg writer synchronously until it finishes.
	if err := ffmpeg.Run(); err != nil {
		return "", ffmpegErr.Wrap(err)
	}

	// Wait for opusenc to finish reading and encoding.
	if err := opusenc.Wait(); err != nil {
		return "", opusencErr.Wrap(err)
	}

	// File is written. Return opusenc's output.
	return opusencErr.stderr.String(), nil
}

// ConvertExt changes a file's extension to the given ext, for example "opus".
func ConvertExt(file string, ext string) string {
	oldExt := filepath.Ext(file)
	// This slicing basically means to slice from start to right before the file
	// extension (so before the dot).
	return file[:len(file)-len(oldExt)] + "." + ext
}
