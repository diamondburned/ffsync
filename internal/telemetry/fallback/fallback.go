package fallback

import (
	"log"
	"time"

	"github.com/diamondburned/ffsync/internal/telemetry"
)

type client struct{}

func New() telemetry.Telemeter {
	return client{}
}

func (client) WriteDuration(dura time.Duration, name string, attrs telemetry.Extras) {
	log.Printf("%s took %v to complete; attrs: %+v\n", name, dura, attrs)
}

func (client) Close() {}
