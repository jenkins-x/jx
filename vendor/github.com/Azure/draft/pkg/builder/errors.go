package builder

import "errors"

var (
	// ErrChartNotExist is returned when no chart/ directory exists during "draft up."
	ErrChartNotExist = errors.New("chart/ does not exist. Please create it using 'draft create' before calling 'draft up'")
	// ErrDockerfileNotExist is returned when no Dockerfile exists during "draft up."
	ErrDockerfileNotExist = errors.New("Dockerfile does not exist. Please create it using 'draft create' before calling 'draft up'")
)
