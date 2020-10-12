package ffmpeg

import (
	"bufio"
	"io"
	"strconv"
	"strings"
	"time"
)

type ProgressType string

const (
	// ProgressContinue is printed when encoding
	ProgressContinue ProgressType = "continue"

	// ProgressEnd is called when encoding is finished
	ProgressEnd ProgressType = "end"

	ProgressUnknown ProgressType = ""
)

type Progress struct {
	Bitrate    float32
	TotalSize  int64
	OutTimeus  int64
	DupFrames  uint64
	DropFrames uint64
	Speed      float32
	Percentage float32
	Progress   ProgressType
}

// OutDuration returns the calculated duration of the output.
func (p Progress) OutDuration() time.Duration {
	return time.Duration(p.OutTimeus) * time.Microsecond
}

// parseOutput parses the output synchronously.
func parseOutput(out io.Reader, callback func(p Progress)) (result string, err error) {
	var progress Progress

	var scanner = bufio.NewScanner(out)
	for scanner.Scan() {
		kv := strings.Split(scanner.Text(), "=")
		if len(kv) != 2 {
			break
		}

		switch kv[0] {
		case "bitrate":
			kv[1] = strings.TrimSuffix(strings.TrimSpace(kv[1]), "kbits/s")
			f, _ := strconv.ParseFloat(kv[1], 32)
			progress.Bitrate = float32(f)
		case "total_size":
			progress.TotalSize, _ = strconv.ParseInt(kv[1], 10, 64)
		case "out_time_us":
			progress.OutTimeus, _ = strconv.ParseInt(kv[1], 10, 64)
		case "dup_frames":
			progress.DupFrames, _ = strconv.ParseUint(kv[1], 10, 64)
		case "drop_frames":
			progress.DropFrames, _ = strconv.ParseUint(kv[1], 10, 64)
		case "speed":
			kv[1] = strings.TrimSuffix(strings.TrimSpace(kv[1]), "x")
			f, _ := strconv.ParseFloat(kv[1], 32)
			progress.Speed = float32(f)
		case "progress":
			progress.Progress = ProgressType(kv[1])
			callback(progress)
		}
	}

	// Spin the rest.
	var finalized strings.Builder
	for scanner.Scan() {
		finalized.WriteString(scanner.Text())
	}

	return finalized.String(), scanner.Err()
}
