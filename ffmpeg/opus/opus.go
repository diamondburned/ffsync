package opus

import (
	"context"

	"github.com/diamondburned/ffsync/ffmpeg"
)

const (
	Bitrate = "128k"
	VBRMode = "constrained"
)

// ConvertCtx atomically converts src to dst as an opus file.
func ConvertCtx(ctx context.Context, src, dst string) (*ffmpeg.Result, error) {
	return ffmpeg.ExecuteCtx(ctx, src, dst,
		// Output format and options
		"-f", "opus", "-vn",
		// Audio encoding options
		"-c:a", "libopus", "-b:a", Bitrate, "-vbr", VBRMode,
	)
}
