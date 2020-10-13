package influx

import (
	"log"
	"sync"
	"time"

	"github.com/diamondburned/ffsync/internal/telemetry"
	"github.com/pkg/errors"

	client "github.com/influxdata/influxdb1-client/v2"
)

type Config struct {
	Database string `env:"FFSYNC_INFLUX_DATABASE"` // default "ffsync"
	Address  string `env:"FFSYNC_INFLUX_ADDRESS"`
	Username string `env:"FFSYNC_INFLUX_USERNAME"`
	Password string `env:"FFSYNC_INFLUX_PASSWORD"`
}

func (c Config) batchPts() client.BatchPoints {
	b, _ := client.NewBatchPoints(client.BatchPointsConfig{
		Database: c.Database,
	})
	return b
}

type Client struct {
	cli client.Client
	cfg Config
	pts chan *client.Point
	cls chan struct{}
	wg  *sync.WaitGroup
}

var _ telemetry.Telemeter = (*Client)(nil)

func NewClient(cfg Config) (telemetry.Telemeter, error) {
	c, err := client.NewHTTPClient(client.HTTPConfig{
		Addr:     cfg.Address,
		Username: cfg.Username,
		Password: cfg.Password,
	})
	if err != nil {
		return nil, errors.Wrap(err, "Failed to make a new Influx client")
	}

	if cfg.Database == "" {
		cfg.Database = "ffsync"
	}

	r, err := c.Query(client.NewQuery("CREATE DATABASE "+cfg.Database, "", ""))
	if err != nil {
		return nil, errors.Wrap(err, "Failed to send query request")
	}
	if r.Err != "" {
		return nil, errors.Wrap(r.Error(), "Failed to create database")
	}

	var pts = make(chan *client.Point, 25)
	var cls = make(chan struct{})

	wg := sync.WaitGroup{}
	wg.Add(1)

	go func() {
		var ticker = time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		defer wg.Done()

		var batch = cfg.batchPts()

		for {
			select {
			case <-cls:
				writeBatch(c, batch) // write before exiting
				return
			case <-ticker.C:
				if len(batch.Points()) > 0 {
					// Write the batch points in a goroutine.
					writeBatch(c, batch)
					// Make a new set of batch points.
					batch = cfg.batchPts()
				}
			case p := <-pts:
				// Add the point into the batch list.
				batch.AddPoint(p)
			}
		}
	}()

	return &Client{
		cli: c,
		cfg: cfg,
		pts: pts,
		cls: cls,
		wg:  &wg,
	}, nil
}

func writeBatch(c client.Client, b client.BatchPoints) {
	if err := c.Write(b); err != nil {
		log.Println("InfluxDB: Failed to write batch points:", err)
	}
}

func (c *Client) WriteDuration(dura time.Duration, name string, attrs telemetry.Extras) {
	var now = time.Now()

	if attrs == nil {
		attrs = map[string]interface{}{}
	}

	attrs["duration"] = dura.Nanoseconds()

	p, err := client.NewPoint(name, nil, attrs, now)
	if err != nil {
		log.Println("BUG: NewPoint errored out:", err)
		return
	}

	c.pts <- p
}

func (c *Client) Close() {
	close(c.cls)
	c.wg.Wait()
	c.cli.Close()
	close(c.pts)
}
