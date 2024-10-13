// Package reload offers lightweight automatic reloading of running processes.
//
// After initialisation with reload.Do() any changes to the binary will restart
// the process.
//
// Example:
//
//    go func() {
//        err := reload.Do(log.Printf)
//        if err != nil {
//            panic(err)
//        }
//    }()
//
// A list of additional directories to watch can be added:
//
//    go func() {
//        err := reload.Do(log.Printf, reload.Dir("tpl", reloadTpl)
//        if err != nil {
//            panic(err)
//        }
//    }()
//
// Note that this package won't prevent race conditions (e.g. when assigning to
// a global templates variable). You'll need to use sync.RWMutex yourself.
package reload // import "github.com/teamwork/reload"

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"
)

var (
	binSelf string

	// The watcher won't be closed automatically, and the file descriptor will be
	// leaked if we don't close it in Exec(); see #9.
	closeWatcher func() error
)

type dir struct {
	path string
	cb   func()
}

// Dir is an additional directory to watch for changes. Directories are watched
// non-recursively.
//
// The second argument is the callback that to run when the directory changes.
// Use reload.Exec() to restart the process.
func Dir(path string, cb func()) dir { return dir{path, cb} }

// Do reload the current process when its binary changes.
//
// The log function is used to display an informational startup message and
// errors. It works well with e.g. the standard log package or Logrus.
//
// The error return will only return initialisation errors. Once initialized it
// will use the log function to print errors, rather than return.
func Do(log func(string, ...interface{}), additional ...dir) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("reload.Do: cannot setup watcher: %w", err)
	}
	closeWatcher = watcher.Close

	timers := make(map[string]*time.Timer)

	binSelf, err = self()
	if err != nil {
		return err
	}
	timers[binSelf] = stoppedTimer(Exec)

	// Watch the directory, because a recompile renames the existing
	// file (rather than rewriting it), so we won't get events for that.
	dirs := make([]string, len(additional)+1)
	dirs[0] = filepath.Dir(binSelf)

	for i, a := range additional {
		path, err := filepath.Abs(a.path)
		if err != nil {
			return fmt.Errorf("reload.Do: cannot get absolute path to %q: %w",
				a.path, err)
		}

		s, err := os.Stat(path)
		if err != nil {
			return fmt.Errorf("reload.Do: %w", err)
		}
		if !s.IsDir() {
			return fmt.Errorf("reload.Do: not a directory: %q; can only watch directories",
				a.path)
		}

		additional[i].path = path
		dirs[i+1] = path
		timers[path] = stoppedTimer(a.cb)
	}

	done := make(chan bool)
	go func() {
		for {
			select {
			case err := <-watcher.Errors:
				if err != nil {
					log("reload error: %v", err)
				}
			case event := <-watcher.Events:
				trigger := (event.Op&fsnotify.Write == fsnotify.Write) || (event.Op&fsnotify.Create == fsnotify.Create)
				if !trigger {
					continue
				}

				if event.Name == binSelf {
					timers[binSelf].Reset(100 * time.Millisecond)
				}

				for _, a := range additional {
					if strings.HasPrefix(event.Name, a.path) {
						timers[a.path].Reset(100 * time.Millisecond)
					}
				}
			}
		}
	}()

	for _, d := range dirs {
		if err := watcher.Add(d); err != nil {
			return fmt.Errorf("reload.Do: cannot add %q to watcher: %w", d, err)
		}
	}

	add := ""
	if len(additional) > 0 {
		reldirs := make([]string, len(dirs)-1)
		for i := range dirs[1:] {
			reldirs[i] = relpath(dirs[i+1])
		}
		add = fmt.Sprintf(" (additional dirs: %s)", strings.Join(reldirs, ", "))
	}
	log("restarting %q when it changes%s", relpath(binSelf), add)
	<-done
	return nil
}

// Exec replaces the current process with a new copy of itself.
func Exec() {
	execName := binSelf
	if execName == "" {
		selfName, err := self()
		if err != nil {
			panic(fmt.Sprintf("cannot restart: cannot find self: %v", err))
		}
		execName = selfName
	}

	if closeWatcher != nil {
		closeWatcher()
	}

	err := syscall.Exec(execName, append([]string{execName}, os.Args[1:]...), os.Environ())
	if err != nil {
		panic(fmt.Sprintf("cannot restart: %v", err))
	}
}

func stoppedTimer(cb func()) *time.Timer {
	t := time.AfterFunc(math.MaxInt64, cb)
	t.Stop()
	return t
}

// Get location to executable.
func self() (string, error) {
	bin := os.Args[0]
	if !filepath.IsAbs(bin) {
		var err error
		bin, err = os.Executable()
		if err != nil {
			return "", fmt.Errorf(
				"cannot get path to binary %q (launch with absolute path): %w",
				os.Args[0], err)
		}
	}
	return bin, nil
}

// Get path relative to cwd.
func relpath(p string) string {
	cwd, err := os.Getwd()
	if err != nil {
		return p
	}

	if strings.HasPrefix(p, cwd) {
		return "./" + strings.TrimLeft(p[len(cwd):], "/")
	}

	return p
}
