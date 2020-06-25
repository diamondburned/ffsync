package telemetry

import "time"

// Exporter is used for errors to extend.
type Exporter interface {
	Export() (attrs map[string]string)
}

type Extras = map[string]interface{}

type Telemeter interface {
	Close()
	Error(err error)
	WriteDuration(dura time.Duration, name string, attrs Extras)
}
