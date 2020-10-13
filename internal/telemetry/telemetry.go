package telemetry

import "time"

type Extras = map[string]interface{}

type Telemeter interface {
	Close()
	WriteDuration(dura time.Duration, name string, attrs Extras)
}

func Batch(ts ...Telemeter) Telemeter {
	return batchTelemeter{
		telemeters: ts,
	}
}

type batchTelemeter struct {
	telemeters []Telemeter
}

func (b batchTelemeter) Close() {
	for _, t := range b.telemeters {
		t.Close()
	}
}

func (b batchTelemeter) WriteDuration(dura time.Duration, name string, attrs Extras) {
	for _, t := range b.telemeters {
		t.WriteDuration(dura, name, attrs)
	}
}
