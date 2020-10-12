package telemetry

import "time"

type Extras = map[string]interface{}

type Telemeter interface {
	Close()
	WriteDuration(dura time.Duration, name string, attrs Extras)
}
