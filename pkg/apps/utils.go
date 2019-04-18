package apps

import (
	"strings"
)

func ToValidFileSystemName(name string) string {
	replacer := strings.NewReplacer(".", "_", "/", "_")
	return replacer.Replace(name)
}
