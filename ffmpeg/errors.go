package ffmpeg

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os/exec"
)

type Error struct {
	wrapped error
	stderr  bytes.Buffer
}

func (err *Error) Error() string {
	return fmtError("ffmpeg failed", err.wrapped, err.stderr)
}

func (err *Error) Unwrap() error {
	return err.wrapped
}

func (err *Error) Wrap(wrapped error) error {
	err.wrapped = wrapped
	return err
}

func (err *Error) Stderr() io.Writer {
	return &err.stderr
}

func (err *Error) StderrBytes() []byte {
	return err.stderr.Bytes()
}

func fmtError(prefix string, err error, stderr bytes.Buffer) string {
	return fmt.Sprintf("%s: %s\n%s", prefix, err.Error(), stderr.String())
}

// ErrIsExit returns true if the error is a process exit non-zero error.
func ErrIsExit(err error) bool {
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		return false
	}

	return exitErr.ExitCode() > 0
}
