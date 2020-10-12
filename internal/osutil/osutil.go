package osutil

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/pkg/errors"
)

// Copy copies file src to dst.
func Copy(ctx context.Context, src, dst string) error {
	// Attempt to hard link for performance.
	if err := os.Link(src, dst); err == nil {
		return nil
	}

	return slowCopy(ctx, src, dst)
}

// slowCopy force copies the content of a file.
func slowCopy(ctx context.Context, src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return errors.Wrap(err, "failed to open src")
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return errors.Wrap(err, "failed to create dst")
	}
	defer dstFile.Close()

	if t, ok := ctx.Deadline(); ok {
		srcFile.SetDeadline(t)
		dstFile.SetDeadline(t)
	}

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return errors.Wrap(err, "failed to copy")
	}

	return nil
}

func MoveTimeout(timeout time.Duration, src, dst string) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	return Move(ctx, src, dst)
}

// Move moves src to dst.
func Move(ctx context.Context, src, dst string) error {
	// Try the fast way.
	if err := os.Rename(src, dst); err == nil {
		return nil
	}

	// Fall back to slow copy. Since rename wouldn't work, hard link wouldn't
	// either.
	if err := slowCopy(ctx, src, dst); err != nil {
		os.Remove(dst)
		return errors.Wrap(err, "failed to copy")
	}

	if err := os.Remove(src); err != nil {
		return errors.Wrap(err, "failed to remove src")
	}

	return nil
}

// RemoveAllIfEmpty removes everything in removeDir, but it removes the
// directory as well if emptyFile's directory has nothing in it anymore.
func RemoveAllIfEmpty(emptyFile, removeDir string) error {
	f, err := os.Open(filepath.Dir(emptyFile))
	if err != nil {
		return errors.Wrap(err, "failed to open")
	}
	defer f.Close()

	// It's likely this isn't a directory if we can't read it, so we can try and
	// remove everything anyway.
	n, err := f.Readdirnames(1)
	if err != nil || len(n) > 0 {
		return os.RemoveAll(removeDir)
	}

	// If this is an empty directory, then remove the other one.
	if len(n) == 0 {
		return os.RemoveAll(filepath.Dir(removeDir))
	}

	// Impossible case.
	return nil
}
