package ffmpeg

import (
	"context"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/pkg/errors"
)

func init() {
	for _, arg0 := range []string{"ffmpeg"} {
		if _, err := exec.LookPath(arg0); err != nil {
			log.Fatalln("Failed to find", arg0+".")
		}
	}
}

// ConvertExt changes a file's extension to the given ext, for example "opus".
func ConvertExt(file string, ext string) string {
	oldExt := filepath.Ext(file)
	// This slicing basically means to slice from start to right before the file
	// extension (so before the dot).
	return file[:len(file)-len(oldExt)] + "." + ext
}

// FileIsExt returns true if the file has the given extension.
func FileIsExt(file string, ext string) bool {
	return strings.HasSuffix(file, "."+ext)
}

var ErrInvalidFileFormat = errors.New("invalid file format")

var defaultArgs = []string{
	"-loglevel", "warning", "-hide_banner", "-threads", "1", "-y", // force yes
}

type Result struct {
	Progress
	Runtime    time.Duration
	OutputPath string
}

// ExecuteCtx executes ffmpeg with the given arguments. It returns the final
// progress result.
func ExecuteCtx(ctx context.Context, src, dst string, args ...string) (*Result, error) {
	// The path to a temporary file, which is basically the same path but with a
	// dot prepended to the filename: /path/to/.file
	tmpdst := filepath.Join(filepath.Dir(dst), "."+filepath.Base(dst))

	// Convert and write to that temp file.
	p, err := executeCtx(ctx, src, tmpdst, args...)
	if err != nil {
		os.Remove(tmpdst)
		return nil, err
	}
	p.OutputPath = dst

	// Atomically rename that temp file to the intended destination.
	return p, os.Rename(tmpdst, dst)
}

func executeCtx(ctx context.Context, src, dst string, args ...string) (*Result, error) {
	ffmpegArgs := make([]string, 0, len(defaultArgs)+len(args)+3)
	ffmpegArgs = append(ffmpegArgs, defaultArgs...)
	ffmpegArgs = append(ffmpegArgs, "-i", src)
	ffmpegArgs = append(ffmpegArgs, args...)
	ffmpegArgs = append(ffmpegArgs, dst)

	cmd := exec.CommandContext(ctx, "ffmpeg", ffmpegArgs...)
	cmd.Env = append(os.Environ(), "AV_LOG_FORCE_NOCOLOR=0") // force no color

	// Use a custom local stderr buffer.
	ffmpegErr := Error{}
	cmd.Stderr = ffmpegErr.Stderr()

	o, err := cmd.StdoutPipe()
	if err != nil {
		return nil, errors.Wrap(err, "failed to open stdout pipe")
	}
	defer o.Close()

	var progress Progress
	var now = time.Now()

	if err := cmd.Start(); err != nil {
		return nil, ffmpegErr.Wrap(err)
	}

	_, err = parseOutput(o, func(p Progress) { progress = p })
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse output")
	}

	if err := cmd.Wait(); err != nil {
		return nil, ffmpegErr.Wrap(err)
	}

	return &Result{
		Progress: progress,
		Runtime:  time.Now().Sub(now),
	}, nil
}
