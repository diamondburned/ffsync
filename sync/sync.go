package sync

import (
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"time"

	"github.com/diamondburned/ffsync/internal/osutil"
	"github.com/pkg/errors"
	"github.com/radovskyb/watcher"
)

type Converter interface {
	QueueCopy(src, dst string)
	QueueConvert(src, dst string)
	ConvertExt(name string) string
}

type Syncer struct {
	w *watcher.Watcher
	c Converter

	path string
	dest string
	opts Options
}

func New(src, dst string, opts Options, c Converter) (*Syncer, error) {
	// Get the source path as an absolute one.
	a, err := filepath.Abs(src)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get the absolute path for given src")
	}

	w := watcher.New()
	w.Event = make(chan watcher.Event, 2) // buffered
	w.FilterOps(watcher.Create, watcher.Move, watcher.Rename, watcher.Remove)

	s := &Syncer{
		w:    w,
		c:    c,
		dest: dst,
		path: a,
		opts: opts,
	}

	w.AddFilterHook(s.checkPath)

	return s, nil
}

// Run starts the watcher and the main loop. It blocks forever. An error is
// returned prematurely, if there is one.
func (s *Syncer) Run(freq time.Duration) error {
	// Prepare the destination directory.
	if err := os.MkdirAll(s.dest, 0775); err != nil {
		return errors.Wrap(err, "Failed to mkdir -p destination directory")
	}

	if err := s.w.AddRecursive(s.path); err != nil {
		return errors.Wrap(err, "Failed to add src recursively")
	}

	defer s.close()

	// Catch up on non-encoded files.
	go filepath.Walk(s.path, func(path string, info os.FileInfo, err error) error {
		// Manually check if this is the right file.
		if s.checkPath(info, path) != nil {
			return nil
		}
		// Send the event into the event channel.
		s.w.Event <- watcher.Event{
			Op:       watcher.Create,
			Path:     path,
			FileInfo: info,
		}
		return nil
	})

	go s.w.Start(freq)

	sig := make(chan os.Signal)
	signal.Notify(sig, os.Interrupt)

	for {
		select {
		case ev := <-s.w.Event:
			s.event(ev)
		case err := <-s.w.Error:
			s.opts.ErrorLog(err)
		case <-s.w.Closed:
			return nil
		case <-sig:
			return nil
		}
	}
}

func (s *Syncer) close() error {
	s.w.Close()
	return nil
}

func (s *Syncer) event(ev watcher.Event) {
	switch ev.Op {
	case watcher.Create:
		dst := s.trans(ev)
		log.Println("Created at", dst)

		// Since there might be a race condition between events being sent,
		// we're best ensuring a directory is made before every single file.
		s.catch(os.MkdirAll(filepath.Dir(dst), 0775), "mkdir -p from create")
		// Well, we should only transcode a file.
		if !ev.IsDir() {
			// Free to interrupt.
			s.OnCreate(ev.Path, dst)
		}

	case watcher.Move:
		src, dst := s.pair(ev)
		log.Println("Moved from", src, "to", dst)
		s.catch(osutil.MoveTimeout(time.Minute, src, dst), "mv")

	case watcher.Rename:
		src, dst := s.pair(ev)
		log.Println("Renamed from", src, "to", dst)
		s.catch(os.Rename(s.pair(ev)), "rename from rename")

	case watcher.Remove:
		dst := s.trans(ev)
		log.Println("Removed", dst)
		s.catch(osutil.RemoveAllIfEmpty(ev.Path, dst), "rm -r from remove")
	}
}

func (s *Syncer) OnCreate(src, dst string) {
	// See if the file already exists in the destination.
	if _, err := os.Stat(dst); err == nil {
		return
	}

	switch s.opts.action(filepath.Ext(src)) {
	case copyAction:
		s.c.QueueCopy(src, dst)
	case convertAction:
		s.c.QueueConvert(src, dst)
	}
}

func (s *Syncer) checkPath(i os.FileInfo, abs string) error {
	// Skip hidden files and directories.
	if strings.HasPrefix(filepath.Base(abs), ".") {
		return watcher.ErrSkip
	}

	// Allow directories.
	if i.IsDir() {
		return nil
	}

	// Allow whitelisted file extensions prefixed with a dot (.)
	if s.opts.IsExt(filepath.Ext(abs)) {
		return nil
	}

	// Skip if neither matched.
	return watcher.ErrSkip
}

// transpath returns the transformed path from the given path
func (s *Syncer) transpath(abs string, dir bool) (path string) {
	// Trim the prefix.
	path = strings.TrimPrefix(abs, s.path)
	// Add the new prefix.
	path = filepath.Join(s.dest, path)
	// If this is not a directory, then convert the extension.
	if !dir {
		path = s.c.ConvertExt(path)
	}
	return path
}

// trans returns the transformed path from the given event
func (s *Syncer) trans(ev watcher.Event) (path string) {
	return s.transpath(ev.Path, ev.IsDir())
}

// pair returns the old and new path relative to the destination path.
func (s *Syncer) pair(ev watcher.Event) (string, string) {
	return s.transpath(ev.OldPath, ev.IsDir()), s.transpath(ev.Path, ev.IsDir())
}

func (s *Syncer) catch(err error, failedTo string) {
	if err != nil {
		s.opts.ErrorLog(errors.Wrap(err, "Failed to "+failedTo))
	}
}
