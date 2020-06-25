package opus

import (
	"bytes"
	"fmt"
	"io"
)

type FFmpegError struct {
	wrapped error
	stderr  bytes.Buffer
}

func (err *FFmpegError) Error() string {
	return fmtError("ffmpeg failed", err.wrapped, err.stderr)
}

func (err *FFmpegError) Wrap(wrapped error) error {
	err.wrapped = wrapped
	return err
}

func (err *FFmpegError) Stderr() io.Writer {
	return &err.stderr
}

type OpusencError struct {
	wrapped error
	stderr  bytes.Buffer
}

func (err *OpusencError) Error() string {
	return fmtError("opusenc failed", err.wrapped, err.stderr)
}

func (err *OpusencError) Wrap(wrapped error) error {
	err.wrapped = wrapped
	return err
}

func (err *OpusencError) Stderr() io.Writer {
	return &err.stderr
}

func fmtError(prefix string, err error, stderr bytes.Buffer) string {
	return fmt.Sprintf("%s: %s\n%s", prefix, err.Error(), stderr.String())
}
