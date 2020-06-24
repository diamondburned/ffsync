package opus

import (
	"bytes"
	"io"

	"github.com/diamondburned/ffsync/telemetry"
	"github.com/pkg/errors"
)

type FFmpegError struct {
	wrapped error
	stderr  bytes.Buffer
}

var _ telemetry.Exporter = (*FFmpegError)(nil)

func (err *FFmpegError) Error() string {
	return "ffmpeg failed: " + err.wrapped.Error() + ": " + err.stderr.String()
}

func (err *FFmpegError) Wrap(wrapped error) error {
	err.wrapped = wrapped
	return err
}

func (err *FFmpegError) Export() (string, map[string]string) {
	return "ffmpeg error", map[string]string{
		"error":  err.wrapped.Error(),
		"stderr": err.stderr.String(),
	}
}

func (err *FFmpegError) Stderr() io.Writer {
	return &err.stderr
}

type OpusencError struct {
	wrapped error
	stderr  bytes.Buffer
}

var _ telemetry.Exporter = (*OpusencError)(nil)

func (err *OpusencError) Error() string {
	return "opusenc failed: " + err.wrapped.Error() + ": " + err.stderr.String()
}

func (err *OpusencError) Wrap(wrapped error) error {
	err.wrapped = errors.Wrap(wrapped, "opusenc failed")
	return err
}

func (err *OpusencError) Export() (string, map[string]string) {
	return "opusenc error", map[string]string{
		"error":  err.Error(),
		"stderr": err.stderr.String(),
	}
}

func (err *OpusencError) Stderr() io.Writer {
	return &err.stderr
}
