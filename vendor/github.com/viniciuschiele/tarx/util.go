package tarx

import (
	"io"
	"os"
	"strings"
)

type readCloserWrapper struct {
	io.ReadCloser
	Reader io.Reader
}

func (rc *readCloserWrapper) Read(p []byte) (n int, err error) {
	return rc.Reader.Read(p)
}

func (rc *readCloserWrapper) Close() error {
	return nil
}

func createFile(filePath string, mode os.FileMode, reader io.Reader) error {
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY, mode)
	if err != nil {
		return err
	}

	defer file.Close()

	if _, err := io.Copy(file, reader); err != nil {
		return err
	}

	return nil
}

func prepareFilters(filters []string) [][]string {
	if filters == nil {
		filters = []string{}
	}

	preparedFilters := make([][]string, len(filters))

	for i, filter := range filters {
		preparedFilters[i] = strings.Split(filter, string(os.PathSeparator))
	}

	return preparedFilters
}

func optimizedMatches(path string, filters [][]string) bool {
	if len(filters) == 0 {
		return true
	}

	pathDirs := strings.Split(path, string(os.PathSeparator))

	for _, filter := range filters {
		i := 0
		count := min(len(pathDirs), len(filter))

		for {
			if i == count {
				return true
			}

			if pathDirs[i] != filter[i] {
				break
			}

			i++
		}
	}

	return false
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
