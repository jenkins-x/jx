package pack

import (
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"k8s.io/helm/pkg/proto/hapi/chart"
)

const testDockerfile = `FROM nginx:latest
`
const testTasksFile = `[pre-up]
pre-up-task = "echo pre-up"

[post-deploy]
setup-task = "echo setup"

[cleanup]
cleanup-task = "echo cleanup"
`

func TestSaveDir(t *testing.T) {
	p := &Pack{
		Chart: &chart.Chart{
			Metadata: &chart.Metadata{
				Name: "chart-for-nigel-thornberry",
			},
		},
		Files: map[string]io.ReadCloser{
			dockerfileName: ioutil.NopCloser(bytes.NewBufferString(testDockerfile)),
			TasksFileName:  ioutil.NopCloser(bytes.NewBufferString(testTasksFile)),
		},
	}
	dir, err := ioutil.TempDir("", "draft-pack-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	if err := p.SaveDir(dir); err != nil {
		t.Errorf("expected there to be no error when writing to %v, got %v", dir, err)
	}

	_, err = os.Stat(filepath.Join(dir, dockerfileName))
	if err != nil {
		if os.IsNotExist(err) {
			t.Errorf("Expected %s to be created but wasn't", dockerfileName)
		} else {
			t.Fatal(err)
		}
	}

	tasksPath := filepath.Join(dir, TargetTasksFileName)
	_, err = os.Stat(tasksPath)
	if err != nil {
		if os.IsNotExist(err) {
			t.Errorf("Expected %s to have been created but wasnt", TargetTasksFileName)
		} else {
			t.Fatal(err)
		}
	}

	data, err := ioutil.ReadFile(tasksPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) == "" {
		t.Error("Expected content in .draft-tasks.toml, got empty string")
	}
}

func TestSaveDirDockerfileExistsInAppDir(t *testing.T) {
	p := &Pack{
		Chart: &chart.Chart{
			Metadata: &chart.Metadata{
				Name: "chart-for-nigel-thornberry",
			},
		},
		Files: map[string]io.ReadCloser{
			dockerfileName: ioutil.NopCloser(bytes.NewBufferString(testDockerfile)),
		},
	}
	dir, err := ioutil.TempDir("", "draft-pack-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	tmpfn := filepath.Join(dir, "Dockerfile")
	expectedDockerfile := []byte("FROM draft")
	if err := ioutil.WriteFile(tmpfn, expectedDockerfile, 0644); err != nil {
		t.Fatal(err)
	}

	if err := p.SaveDir(dir); err != nil {
		t.Errorf("expected there to be no error when writing to %v, got %v", dir, err)
	}

	savedDockerfile, err := ioutil.ReadFile(tmpfn)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(savedDockerfile, expectedDockerfile) {
		t.Errorf("expected '%s', got '%s'", string(expectedDockerfile), string(savedDockerfile))
	}
}
