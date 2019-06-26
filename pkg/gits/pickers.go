package gits

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"gopkg.in/AlecAivazis/survey.v1"
	"gopkg.in/AlecAivazis/survey.v1/terminal"
)

func PickGitRepoName(batchMode, allowExistingRepo bool, provider GitProvider, defaultRepoName, owner string, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) (string, error) {
	surveyOpts := survey.WithStdio(in, out, errOut)
	repoName := ""
	if batchMode {
		repoName = defaultRepoName
		if repoName == "" {
			repoName = "dummy"
		}
	} else {
		prompt := &survey.Input{
			Message: "Enter the new repository name: ",
			Default: defaultRepoName,
		}
		validator := func(val interface{}) error {
			str, ok := val.(string)
			if !ok {
				return fmt.Errorf("Expected string value")
			}
			if strings.TrimSpace(str) == "" {
				return fmt.Errorf("Repository name is required")
			}
			if allowExistingRepo {
				return nil
			}
			return provider.ValidateRepositoryName(owner, str)
		}
		err := survey.AskOne(prompt, &repoName, validator, surveyOpts)
		if err != nil {
			return "", err
		}
		if repoName == "" {
			return "", fmt.Errorf("No repository name specified")
		}
	}
	return repoName, nil
}

func PickGitRepoOwner(batchMode bool, provider GitProvider, gitUsername string, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) (string, error) {
	owner := ""
	if batchMode {
		owner = gitUsername
	} else {
		org, err := PickOrganisation(provider, gitUsername, in, out, errOut)
		if err != nil {
			return "", err
		}
		owner = org
		if org == "" {
			owner = gitUsername
		}
	}
	return owner, nil
}

func PickRepositories(provider GitProvider, owner string, message string, selectAll bool, filter string, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) ([]*GitRepository, error) {
	answer := []*GitRepository{}
	repos, err := provider.ListRepositories(owner)
	if err != nil {
		return answer, err
	}

	repoMap := map[string]*GitRepository{}
	allRepoNames := []string{}
	for _, repo := range repos {
		n := repo.Name
		if n != "" && (filter == "" || strings.Contains(n, filter)) {
			allRepoNames = append(allRepoNames, n)
			repoMap[n] = repo
		}
	}
	if len(allRepoNames) == 0 {
		return answer, fmt.Errorf("No matching repositories could be found!")
	}
	sort.Strings(allRepoNames)

	prompt := &survey.MultiSelect{
		Message: message,
		Options: allRepoNames,
	}
	if selectAll {
		prompt.Default = allRepoNames
	}
	repoNames := []string{}
	surveyOpts := survey.WithStdio(in, out, errOut)
	err = survey.AskOne(prompt, &repoNames, nil, surveyOpts)

	for _, n := range repoNames {
		repo := repoMap[n]
		if repo != nil {
			answer = append(answer, repo)
		}
	}
	return answer, err
}

func PickOrganisation(orgLister OrganisationLister, userName string, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) (string, error) {
	prompt := &survey.Select{
		Message: "Which organisation do you want to use?",
		Options: GetOrganizations(orgLister, userName),
		Default: userName,
	}

	orgName := ""
	surveyOpts := survey.WithStdio(in, out, errOut)
	err := survey.AskOne(prompt, &orgName, nil, surveyOpts)
	if err != nil {
		return "", err
	}
	if orgName == userName {
		return "", nil
	}
	return orgName, nil
}
