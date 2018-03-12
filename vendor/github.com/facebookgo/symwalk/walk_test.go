package symwalk

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/facebookgo/ensure"
	"github.com/facebookgo/testname"
)

type WalkFn struct {
	List []string
}

func (w *WalkFn) walkFn(path string, info os.FileInfo, err error) error {
	w.List = append(w.List, path)
	return nil
}

func makeEmptyRoot(t *testing.T) string {
	prefix := fmt.Sprintf("%s-", testname.Get("walk-"))
	root, err := ioutil.TempDir("", prefix)
	ensure.Nil(t, err)
	return root
}

func createFile(t *testing.T, path string) {
	file, err := os.Create(path)
	ensure.Nil(t, err)
	ensure.Nil(t, file.Close())
}

func TestSingleFile(t *testing.T) {
	fn := new(WalkFn)
	root := makeEmptyRoot(t)
	defer os.RemoveAll(root)

	createFile(t, filepath.Join(root, "a"))

	ensure.Nil(t, Walk(root, fn.walkFn))

	ensure.DeepEqual(t,
		fn.List,
		[]string{
			root,
			filepath.Join(root, "a"),
		},
	)
}

func TestSingleLinkedFile(t *testing.T) {
	fn := new(WalkFn)
	root := makeEmptyRoot(t)
	defer os.RemoveAll(root)

	createFile(t, filepath.Join(root, "a"))
	ensure.Nil(t, os.Symlink(filepath.Join(root, "a"), filepath.Join(root, "link_a")))

	ensure.Nil(t, Walk(root, fn.walkFn))

	ensure.DeepEqual(t,
		fn.List,
		[]string{
			root,
			filepath.Join(root, "a"),
			filepath.Join(root, "link_a"),
		},
	)
}

func TestLinkedDir(t *testing.T) {
	fn := new(WalkFn)
	root := makeEmptyRoot(t)
	defer os.RemoveAll(root)

	ensure.Nil(t, os.Mkdir(filepath.Join(root, "a"), 0755))

	createFile(t, filepath.Join(root, "a", "1"))
	ensure.Nil(t, os.Symlink(filepath.Join(root, "a"), filepath.Join(root, "b")))

	ensure.Nil(t, Walk(root, fn.walkFn))

	ensure.DeepEqual(t, fn.List,
		[]string{
			root,
			filepath.Join(root, "a"),
			filepath.Join(root, "a", "1"),
			filepath.Join(root, "b"),
			filepath.Join(root, "b", "1"),
		},
	)
}

func TestGeneralCase(t *testing.T) {
	fn := new(WalkFn)
	root := makeEmptyRoot(t)
	defer os.RemoveAll(root)

	ensure.Nil(t, os.MkdirAll(filepath.Join(root, "a", "a1"), 0755))
	createFile(t, filepath.Join(root, "a", "a1", "1"))

	ensure.Nil(t, os.Mkdir(filepath.Join(root, "b"), 0755))
	createFile(t, filepath.Join(root, "b", "2"))

	ensure.Nil(t, os.Symlink(filepath.Join(root, "b"), filepath.Join(root, "a", "a1", "a2")))

	ensure.Nil(t, os.Symlink(filepath.Join(root, "a", "a1", "1"), filepath.Join(root, "top")))

	ensure.Nil(t, Walk(root, fn.walkFn))

	ensure.DeepEqual(t,
		fn.List,
		[]string{
			root,
			filepath.Join(root, "a"),
			filepath.Join(root, "a", "a1"),
			filepath.Join(root, "a", "a1", "1"),
			filepath.Join(root, "a", "a1", "a2"),
			filepath.Join(root, "a", "a1", "a2", "2"),
			filepath.Join(root, "b"),
			filepath.Join(root, "b", "2"),
			filepath.Join(root, "top"),
		},
	)
}
