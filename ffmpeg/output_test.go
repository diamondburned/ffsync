package ffmpeg

import (
	"reflect"
	"strings"
	"testing"
)

const testOutput = `bitrate= 104.0kbits/s
total_size=1048576
out_time_us=80313500
out_time_ms=80313500
out_time=00:01:20.313500
dup_frames=0
drop_frames=0
speed=40.1x
progress=continue
bitrate= 29.5kbits/s
total_size=1357046
out_time_us=83853500
out_time_ms=83853500
out_time=00:01:23.853500
dup_frames=0
drop_frames=0
speed=  40x
progress=end
`

func TestParseOutput(t *testing.T) {
	var outputs = []Progress{
		{
			Bitrate:    104.0,
			TotalSize:  1048576,
			OutTimeus:  80313500,
			DupFrames:  0,
			DropFrames: 0,
			Speed:      40.1,
			Progress:   ProgressContinue,
		},
		{
			Bitrate:    29.5,
			TotalSize:  1357046,
			OutTimeus:  83853500,
			DupFrames:  0,
			DropFrames: 0,
			Speed:      40,
			Progress:   ProgressEnd,
		},
	}

	var index int

	r, err := parseOutput(strings.NewReader(testOutput), func(p Progress) {
		if !reflect.DeepEqual(p, outputs[index]) {
			t.Errorf("Mismatch:\nExpect:\t\t%#v\nGot:\t\t%#v", p, outputs[index])
		}

		index++
	})

	if err != nil {
		t.Fatal("Failed to parse output:", err)
	}

	if r != "" {
		t.Fatalf("Unexpected tail: %#v", r)
	}
}
