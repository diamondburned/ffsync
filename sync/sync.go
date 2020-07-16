package sync

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/radovskyb/watcher"
	"golang.org/x/sync/semaphore"
)

type Converter interface {
	ConvertCtx(ctx context.Context, src, dst string) error
	ConvertExt(name string) string
}

type Syncer struct {
	Wait time.Duration // default 1 min

	w *watcher.Watcher
	c Converter

	path string
	dest string
	fmts []string
	sema *semaphore.Weighted

	Error func(err error)
}

func New(src, dst string, filefmts []string, c Converter) (*Syncer, error) {
	// Get the source path as an absolute one.
	a, err := filepath.Abs(src)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get the absolute path for given src")
	}

	w := watcher.New()
	w.Event = make(chan watcher.Event, 100) // buffered
	w.FilterOps(watcher.Create, watcher.Move, watcher.Rename, watcher.Remove)

	s := &Syncer{
		Wait: time.Minute,
		w:    w,
		c:    c,
		dest: dst,
		path: a,
		fmts: filefmts,
		sema: semaphore.NewWeighted(int64(runtime.GOMAXPROCS(-1) / 2)),
		Error: func(err error) {
			log.Println("Error watching:", err)
		},
	}

	w.AddFilterHook(s.pathchk)

	return s, nil
}

func (s *Syncer) Start(freq time.Duration) error {
	// Prepare the destination directory.
	if err := os.MkdirAll(s.dest, 0775); err != nil {
		return errors.Wrap(err, "Failed to mkdir -p destination directory")
	}

	go func() {
		for {
			select {
			case ev := <-s.w.Event:
				s.event(ev)
			case err := <-s.w.Error:
				s.Error(err)
			case <-s.w.Closed:
				return
			}
		}
	}()

	if err := s.w.AddRecursive(s.path); err != nil {
		return errors.Wrap(err, "Failed to add src recursively")
	}

	// Catch up on non-encoded files.
	go filepath.Walk(s.path, func(path string, info os.FileInfo, err error) error {
		// Manually check if this is the right file.
		if s.pathchk(info, path) != nil {
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
	return nil
}

func (s *Syncer) Close() error {
	s.w.Close()
	return nil
}

func (s *Syncer) event(ev watcher.Event) {
	switch ev.Op {
	case watcher.Create:
		// Since there might be a race condition between events being sent,
		// we're best ensuring a directory is made before every single file.
		s.catch(os.MkdirAll(filepath.Dir(s.trans(ev)), 0775), "mkdir -p from create")
		// Well, we should only transcode a file.
		if !ev.IsDir() {
			// Free to interrupt.
			s.transcode(ev.Path, s.trans(ev))
		}

	case watcher.Move:
		s.catch(os.Rename(s.pair(ev)), "rename from move")

	case watcher.Rename:
		s.catch(os.Rename(s.pair(ev)), "rename from rename")

	case watcher.Remove:
		s.catch(os.RemoveAll(s.trans(ev)), "rm -r from remove")
	}
}

func (s *Syncer) transcode(src, dst string) {
	// See if the file already exists in the destination.
	if _, err := os.Stat(dst); err == nil {
		return
	}

	time.AfterFunc(s.Wait, func() {
		// 10 minutes timeout.
		ctx, cancel := context.WithTimeout(context.TODO(), 10*time.Minute)
		defer cancel()

		if err := s.sema.Acquire(ctx, 1); err != nil {
			s.catch(err, "acquire ctx for transcode from create")
			return
		}

		if err := s.c.ConvertCtx(ctx, src, dst); err != nil {
			s.catch(err, "transcode from create")
		}

		s.sema.Release(1)
	})
}

func (s *Syncer) pathchk(i os.FileInfo, abs string) error {
	// Skip hidden files and directories.
	if strings.HasPrefix(filepath.Base(abs), ".") {
		return watcher.ErrSkip
	}

	// Allow directories.
	if i.IsDir() {
		return nil
	}

	// Allow whitelisted file extensions prefixed with a dot (.)
	var ext = filepath.Ext(abs)

	for _, fmt := range s.fmts {
		if fmt == ext {
			return nil
		}
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
		s.Error(errors.Wrap(err, "Failed to "+failedTo))
	}
}
