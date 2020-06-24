package sync

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/diamondburned/ffsync/opus"
)

const prepared = 25
const tick = 100 * time.Millisecond

func TestSyncer(t *testing.T) {
	src := mktmpdir(t)
	dst := mktmpdir(t)

	t.Log("Source directory:", src)
	t.Log("Destination directory:", dst)

	m := newMock(t, src)

	s, err := New(src, dst, []string{".ff"}, m)
	if err != nil {
		t.Fatal("Failed to create syncer:", err)
	}

	s.Error = func(err error) {
		t.Error("Syncer error:", err)
	}

	var runerr = make(chan error)
	go func() {
		runerr <- s.Run(tick)
	}()
	defer s.Close()

	select {
	case err := <-runerr:
		t.Fatal(err)
	case <-time.After(tick * 2):
		// probably no more errors.
	}

	// Wait until we have enough files, with a bit of overhead.
	var timeout = time.After(tick * (prepared + 10))

	var converted = make([]string, 0, prepared)

FileLoop:
	for {
		select {
		case dst := <-m.converted:
			t.Log("Received", dst)
			converted = append(converted, dst)

			// If we have enough files received, then break out of the loop..
			if len(converted) == prepared {
				break FileLoop
			}

		case <-timeout:
			t.Fatal("Timed out waiting for files.")
		}
	}

	// Check that filenames are expected.
	for _, dst := range converted {
		// Filenames are expected to have the new extension.
		if !strings.HasSuffix(dst, ".converted") {
			t.Fatalf("File %s does not have .converted", dst)
		}
	}

	// Try making a new file.
	t.Log("touch test.ff")
	if _, err := os.Create(filepath.Join(m.src, "test.ff")); err != nil {
		t.Fatal("Failed to create an after-the-fact test file:", err)
	}

	// Expect the file and check its name.
	if conv := <-m.converted; filepath.Base(conv) != "test.converted" {
		t.Fatal("New file does not have expected name:", conv)
	}

	// Try making a file with no extension. This should not be converted.
	t.Log("touch test")
	if _, err := os.Create(filepath.Join(m.src, "test")); err != nil {
		t.Fatal("Failed to create an after-the-fact non-test file:", err)
	}

	select {
	case dst := <-m.converted:
		t.Fatal("Unexpected file converted: " + dst)
	case <-time.After(time.Second):
		// Expect the case to timeout.
		break
	}

	// Try making a file in a folder.
	t.Log("mkdir astolfo")
	if err := os.Mkdir(filepath.Join(m.src, "astolfo"), 0750); err != nil {
		t.Fatal("Failed to make a test directory:", err)
	}

	// Make a test file inside that directory.
	t.Log("touch astolfo/test.ff")
	if _, err := os.Create(filepath.Join(m.src, "astolfo", "test.ff")); err != nil {
		t.Fatal("Failed to make an after-the-fact test file in testdir:", err)
	}

	// Expect the file inside the directory.
	if conv := <-m.converted; !strings.HasSuffix(conv, "/astolfo/test.converted") {
		t.Fatal("New file is not in expected location:", conv)
	}

	// Remove the directory, expect no errors.
	if err := os.RemoveAll(filepath.Join(m.src, "astolfo")); err != nil {
		t.Fatal("Failed to remove astolfo/:", err)
	}

	time.Sleep(2 * time.Second)

	// Wait for everything to finish.
	m.wg.Wait()
}

type mock struct {
	converted chan string
	src       string
	wg        sync.WaitGroup
}

func newMock(t *testing.T, src string) *mock {
	// Make dummy files for the syncer to pick up on.
	for i := 0; i < prepared; i++ {
		// Make a file with a fake file format (.ff).
		_, err := os.Create(filepath.Join(src, fmt.Sprintf("test_%d.ff", i)))
		if err != nil {
			t.Fatal("Failed to touch", i)
		}
	}

	return &mock{src: src, converted: make(chan string)}
}

func (m *mock) ConvertCtx(ctx context.Context, src, dst string) error {
	m.wg.Add(1)
	defer m.wg.Done()

	_, err := os.Create(dst)
	if err != nil {
		return err
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(tick):
		// noop
	}

	m.converted <- dst
	return nil
}

func (m *mock) ConvertExt(name string) string {
	return opus.ConvertExt(name, "converted")
}

func mktmpdir(t *testing.T) string {
	p, err := ioutil.TempDir(os.TempDir(), "sync-test-")
	if err != nil {
		t.Fatal("Failed to mktemp:", err)
	}

	t.Cleanup(func() { os.RemoveAll(p) })

	return p
}
