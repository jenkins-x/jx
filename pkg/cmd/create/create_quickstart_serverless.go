package create

import (
	"os/exec"
	"strings"

	"github.com/jenkins-x/jx/pkg/quickstarts"
	"github.com/pkg/errors"
)

// ExecCommand allows easy fakes of `exec.command``
var ExecCommand = exec.Command

// GetServerlessQuickstarts returns the templates supported by `serverless` CLI`
var GetServerlessQuickstarts = func() (*quickstarts.QuickstartModel, error) {
	model := quickstarts.NewQuickstartModel()
	cmd := ExecCommand("serverless", "create", "--help")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return model, err
	}
	lines := strings.Split(string(out), "\n")
	templateLine := ""
	for _, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "--template") {
			templateLine = line
			break
		}
	}
	if len(templateLine) == 0 {
		return model, errors.New("The output does not contain the line starting with `--template`\n" + string(out))
	}
	templates := []string{}
	for _, template := range strings.Split(templateLine, ",") {
		if strings.Contains(template, " and ") {
			templates = append(templates, strings.Split(template, " and ")[0], strings.Split(template, " and ")[1])
		} else {
			templates = append(templates, template)
		}
	}
	for _, template := range templates {
		template = strings.ReplaceAll(template, "\"", "")
		template = strings.TrimSpace(template)
		if strings.HasPrefix(template, "--template") {
			searchString := "Available templates:"
			index := strings.Index(template, searchString)
			if index == -1 {
				return model, errors.New("Could not find `Available templates`")
			}
			template = template[index+len(searchString)+1 : len(template)]
		}
		index := strings.Index(template, "-")
		if index > 0 {
			framework := template[:index]
			language := template[index+1:]
			q := quickstarts.Quickstart{
				ID:        template,
				Name:      template,
				Framework: framework,
				Language:  language,
			}
			model.Add(&q)
		}
	}
	return model, nil
}

// CreateServerlessQuickstart creates a quickstart project
var CreateServerlessQuickstart = func(qf *quickstarts.QuickstartForm, dir string) error {
	ExecCommand("serverless")
	println("serverless " + "create " + "--template " + qf.Quickstart.ID + " --name " + qf.Name + " --path " + qf.Name)
	cmd := ExecCommand("serverless", "create", "--template", qf.Quickstart.ID, "--name", qf.Name, "--path", qf.Name)
	// out, err := cmd.CombinedOutput()
	_, err := cmd.CombinedOutput()
	if err != nil {
		return err
	}
	return nil
}

// func (o *CreateQuickstartOptions) createQuickstart(f *quickstarts.QuickstartForm, dir string) (string, error) {
// 	q := f.Quickstart
// 	answer := filepath.Join(dir, f.Name)
// 	u := q.DownloadZipURL
// 	println(u)
// 	println(dir)
// 	if u == "" {
// 		return answer, fmt.Errorf("quickstart %s does not have a download zip URL", q.ID)
// 	}
// 	client := http.Client{}

// 	req, err := http.NewRequest(http.MethodGet, u, strings.NewReader(""))
// 	if err != nil {
// 		return answer, err
// 	}
// 	userAuth := q.GitProvider.UserAuth()
// 	token := userAuth.ApiToken
// 	username := userAuth.Username
// 	if token != "" && username != "" {
// 		log.Logger().Debugf("Downloading Quickstart source zip from %s with basic auth for user: %s", u, username)
// 		req.SetBasicAuth(username, token)
// 	}
// 	res, err := client.Do(req)
// 	if err != nil {
// 		return answer, err
// 	}
// 	body, err := ioutil.ReadAll(res.Body)
// 	if err != nil {
// 		return answer, err
// 	}

// 	zipFile := filepath.Join(dir, "source.zip")
// 	err = ioutil.WriteFile(zipFile, body, util.DefaultWritePermissions)
// 	if err != nil {
// 		return answer, fmt.Errorf("failed to download file %s due to %s", zipFile, err)
// 	}
// 	tmpDir, err := ioutil.TempDir("", "jx-source-")
// 	if err != nil {
// 		return answer, fmt.Errorf("failed to create temporary directory: %s", err)
// 	}
// 	err = util.Unzip(zipFile, tmpDir)
// 	if err != nil {
// 		return answer, fmt.Errorf("failed to unzip new project file %s due to %s", zipFile, err)
// 	}
// 	err = os.Remove(zipFile)
// 	if err != nil {
// 		return answer, err
// 	}
// 	tmpDir, err = findFirstDirectory(tmpDir)
// 	if err != nil {
// 		return answer, fmt.Errorf("failed to find a directory inside the source download: %s", err)
// 	}
// 	err = util.RenameDir(tmpDir, answer, false)
// 	if err != nil {
// 		return answer, fmt.Errorf("failed to rename temp dir %s to %s: %s", tmpDir, answer, err)
// 	}
// 	log.Logger().Infof("Generated quickstart at %s", answer)
// 	return answer, nil
// }
