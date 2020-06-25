package fallback

import (
	"log"
	"time"

	"github.com/diamondburned/ffsync/telemetry"
)

type Client struct{}

var _ telemetry.Telemeter = (*Client)(nil)

func New() Client {
	return Client{}
}

func (Client) WriteDuration(dura time.Duration, name string, attrs map[string]interface{}) {
	log.Printf("%s took %v to complete; attrs: %+v\n", name, dura, attrs)
}

func (Client) Close() {}
