package parse

import "testing"

const output = `Encoding complete
-----------------------------------------------------
       Encoded: 4 minutes and 31.64 seconds
       Runtime: 4 seconds
                (67.91x realtime)
         Wrote: 3853633 bytes, 13582 packets, 275 pages
       Bitrate: 109.64 kbit/s (without overhead)
 Instant rates: 1.2 to 193.2 kbit/s
                (3 to 483 bytes per packet)
      Overhead: 3.39% (container+metadata)`

func TestParse(t *testing.T) {
	o, err := Parse(output)
	if err != nil {
		t.Fatal(err)
	}

	if o.Encoded.String() != "4m31.64s" {
		t.Fatal("Unexpected encoded duration:", o.Encoded)
	}

	if o.Runtime.String() != "4s" {
		t.Fatal("Unexpected runtime duration:", o.Runtime)
	}
}
