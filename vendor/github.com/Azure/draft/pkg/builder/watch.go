package builder

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/rjeczalik/notify"
	"golang.org/x/net/context"
	"k8s.io/helm/pkg/ignore"
)

const ignoreFileName = ".draftignore"

// Watch watches for inotify events in the build context's application directory, returning events
// to the stream
func (buildctx *Context) Watch(ctx context.Context, stream chan<- *Context) (err error) {
	var rules *ignore.Rules
	if rules, err = ignore.ParseFile(ignoreFileName); err != nil {
		// only fail if exists and can't be parsed
		if _, err = os.Stat(ignoreFileName); os.IsExist(err) {
			return fmt.Errorf("could not load ignore watch list: %v", err)
		}
	}
	defer close(stream)
	return watch(ctx, buildctx.AppDir, rules, func() error {
		b, err := LoadWithEnv(buildctx.AppDir, buildctx.EnvName)
		if err != nil {
			return err
		}
		stream <- b
		return nil
	})
}

func watch(ctx context.Context, dir string, rules *ignore.Rules, action func() error) error {
	infoc := make(chan notify.EventInfo, 1)
	if err := notify.Watch(dir, infoc, notify.All); err != nil {
		return fmt.Errorf("could not watch %q: %v", dir, err)
	}

	evtc := make(chan struct{})
	go func() {
		for info := range infoc {
			prefix := strings.TrimPrefix(info.Path(), dir+"/")
			fi, err := os.Stat(info.Path())
			if os.IsNotExist(err) {
				// create dummy file info for removed file or directory
				fi = removedFileInfo(filepath.Base(info.Path()))
			}
			// only rebuild if the changed file isn't in our ignore list
			if rules != nil && rules.Ignore(prefix, fi) {
				continue
			}
			// ignore manually everything inside the .git/ directory as
			// helm ignore file doesn't have directory and whole content
			// (subdir of subdir) ignore support yet.
			if filepath.HasPrefix(prefix, ".git/") {
				continue
			}
			evtc <- struct{}{}
		}
		close(evtc)
	}()
	defer func() {
		notify.Stop(infoc)
		close(evtc)
	}()
	for {
		select {
		case <-evtc:
			if err := action(); err != nil {
				return err
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// removedFileInfo fake file info for ignore library only use IsDir() in negative pattern
type removedFileInfo string

func (n removedFileInfo) Name() string     { return string(n) }
func (removedFileInfo) Size() int64        { return 0 }
func (removedFileInfo) Mode() os.FileMode  { return 0 }
func (removedFileInfo) ModTime() time.Time { return time.Time{} }
func (removedFileInfo) IsDir() bool        { return false }
func (removedFileInfo) Sys() interface{}   { return nil }

// r, err := ignore.ParseFile(ignoreFileName)
// if err != nil {
// 	// only fail if file can't be parsed but exists
// 	if _, err := os.Stat(ignoreFileName); os.IsExist(err) {
// 		log.Fatalf("could not load ignore watch list %v", err)
// 	}
// }
