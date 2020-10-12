package osutil

import (
	"io"
	"os"

	"github.com/pkg/errors"
)

// Copy copies file src to dst.
func Copy(src, dst string) error {
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

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return errors.Wrap(err, "failed to copy")
	}

	return nil
}
