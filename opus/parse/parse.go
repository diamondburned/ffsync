package parse

import (
	"strconv"
	"strings"
	"time"

	"github.com/diamondburned/sfmatch"
	"github.com/pkg/errors"
)

type Opusenc struct {
	Encoded    time.Duration
	EncodedRaw string `sfmatch:"Encoded: (.+$)"`
	Runtime    time.Duration
	RuntimeRaw string `sfmatch:"Runtime: (.+$)"`

	RealtimeMult float64 `sfmatch:"\\((.+)x realtime\\)"`
	WroteBytes   uint64  `sfmatch:"Wrote: (\\d+) bytes"`
	Bitrate      float64 `sfmatch:"Bitrate: (.+) kbit/s \\(without overhead\\)"`
	Overhead     float64 `sfmatch:"Overhead: (.+)% \\(container\\+metadata\\)"`
}

var match = sfmatch.MustCompile((*Opusenc)(nil))

func Parse(output string) (*Opusenc, error) {
	var enc Opusenc
	if err := match.Unmarshal(output, &enc); err != nil {
		return nil, err
	}

	e, err := parseDuration(enc.EncodedRaw)
	if err != nil {
		return &enc, errors.Wrap(err, "Failed to parse encoded duration")
	}

	r, err := parseDuration(enc.RuntimeRaw)
	if err != nil {
		return &enc, errors.Wrap(err, "Failed to parse runtime duration")
	}

	enc.Encoded = e
	enc.Runtime = r

	return &enc, nil
}

func parseDuration(dura string) (time.Duration, error) {
	var hours, mins time.Duration
	var secs float64
	var i float64

	for _, word := range strings.Fields(dura) {
		switch word {
		case "hour", "hours":
			hours = time.Duration(i)

		case "minute", "minutes":
			mins = time.Duration(i)

		case "second", "seconds":
			secs = i

		case "and":
			continue

		default:
			f, err := strconv.ParseFloat(word, 64)
			if err != nil {
				return 0, err
			}
			i = f
		}
	}

	return 0 +
		(hours * time.Hour) +
		(mins * time.Minute) +
		(time.Duration(secs * float64(time.Second))), nil
}
