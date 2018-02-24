package decorators

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/antham/chyle/chyle/convh"
)

type shellConfig map[string]struct {
	COMMAND string
	ORIGKEY string
	DESTKEY string
}

// shell pipes a shell command on field content and dump the
// result into a new field
type shell struct {
	COMMAND string
	ORIGKEY string
	DESTKEY string
}

func (s shell) Decorate(commitMap *map[string]interface{}) (*map[string]interface{}, error) {
	var tmp interface{}
	var value string
	var ok bool
	var err error

	if tmp, ok = (*commitMap)[s.ORIGKEY]; !ok {
		return commitMap, nil
	}

	if value, err = convh.ConvertToString(tmp); err != nil {
		return commitMap, nil
	}

	if (*commitMap)[s.DESTKEY], err = s.execute(value); err != nil {
		return commitMap, err
	}

	return commitMap, nil
}

func (s shell) execute(value string) (string, error) {
	var result []byte
	var err error

	command := fmt.Sprintf(`echo "%s"|%s`, strings.Replace(value, `"`, `\"`, -1), s.COMMAND)

	/* #nosec */
	if result, err = exec.Command("sh", "-c", command).Output(); err != nil {
		return "", fmt.Errorf("%s : command failed", command)
	}

	return string(result[:len(result)-1]), nil
}

func newShell(configs shellConfig) []Decorater {
	results := []Decorater{}

	for _, config := range configs {
		results = append(results, shell(config))
	}

	return results
}
