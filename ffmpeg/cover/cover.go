package cover

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/diamondburned/ffsync/ffmpeg"
)

const (
	CoverArtSz = "600"
	CoverArtQ  = "5"
)

var (
	vf = fmt.Sprintf("scale=-1:'min(%s,ih)'", CoverArtSz)
)

// ExistsAlbum returns true if the given output path contains a cover.jpg.
func ExistsAlbum(dst string) (string, bool) {
	// Force override the output.
	dst = forceCoverFile(dst)

	s, err := os.Stat(dst)
	if err != nil {
		return dst, false
	}

	return dst, !s.IsDir()
}

// ExtractAlbum takes the art from the src file and extracts its album art into
// cover.jpg. The given dst is the destination to the music file, which this
// function will automatically derive the path to cover.jpg.
func ExtractAlbum(ctx context.Context, src, dst string) (*ffmpeg.Result, error) {
	return ffmpeg.ExecuteCtx(ctx, src, forceCoverFile(dst),
		// Album art options
		"-c:v", "mjpeg", "-sws_flags", "lanczos", "-q:v", CoverArtQ, "-vf", vf,
	)
}

func forceCoverFile(dst string) string {
	if filepath.Base(dst) != "cover.jpg" {
		return filepath.Join(filepath.Dir(dst), "cover.jpg")
	}
	return dst
}

var noStreamErr = []byte("does not contain any stream")

// ErrIsNoStream returns true if the error is FFmpeg saying there isn't a video
// stream. This is useful because not all songs have album arts.
func ErrIsNoStream(err error) bool {
	var ffErr *ffmpeg.Error
	if errors.As(err, &ffErr) {
		return bytes.Contains(ffErr.StderrBytes(), noStreamErr)
	}
	return false
}
