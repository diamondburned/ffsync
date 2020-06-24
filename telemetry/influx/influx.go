package influx

import (
	"time"

	"github.com/diamondburned/ffsync/telemetry"
	influxdb2 "github.com/influxdata/influxdb-client-go"
	"github.com/influxdata/influxdb-client-go/api"
	"github.com/influxdata/influxdb-client-go/api/write"
	"github.com/pkg/errors"
)

type Config struct {
	Database string `env:"FFSYNC_INFLUX_DATABASE"` // default "ffsync"
	Address  string `env:"FFSYNC_INFLUX_ADDRESS"`
	Token    string `env:"FFSYNC_INFLUX_TOKEN"`
}

type Client struct {
	influxdb2.Client
	wr api.WriteApi
}

var _ telemetry.Telemeter = (*Client)(nil)

func NewClient(cfg Config) *Client {
	client := influxdb2.NewClient(cfg.Address, cfg.Token)

	if cfg.Database == "" {
		cfg.Database = "ffsync"
	}

	return &Client{client, client.WriteApi("", cfg.Database)}
}

func (c *Client) Error(err error) {
	var now = time.Now()

	var fields map[string]interface{}
	var name = "error"

	// See if we could also get some more info.
	if exporter, ok := err.(telemetry.Exporter); ok {
		n, t := exporter.Export()
		name = n
		fields = make(map[string]interface{}, len(t))

		for k, v := range t {
			fields[k] = v
		}
	} else {
		fields = make(map[string]interface{}, 2)
	}

	fields["error"] = err.Error()

	// See if we could get a stack trace.
	if st, ok := err.(interface{ StackTrace() errors.StackTrace }); ok {
		if traces := st.StackTrace(); len(traces) > 0 {
			// Grab the first in stack.
			frame, _ := traces[0].MarshalText()
			fields["caller"] = string(frame)
		}
	}

	c.wr.WritePoint(write.NewPoint(name, nil, fields, now))
}

func (c *Client) WriteDuration(dura time.Duration, name string, attrs map[string]interface{}) {
	var now = time.Now()

	if attrs == nil {
		attrs = map[string]interface{}{}
	}

	attrs["duration"] = dura.Nanoseconds()

	c.wr.WritePoint(write.NewPoint(name, nil, attrs, now))
}
