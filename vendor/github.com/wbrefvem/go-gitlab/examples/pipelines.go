package main

import (
	"log"

	"github.com/xanzy/go-gitlab"
)

func pipelineExample() {
	git := gitlab.NewClient(nil, "yourtokengoeshere")
	git.SetBaseURL("https://gitlab.com/api/v4")

	opt := &gitlab.ListProjectPipelinesOptions{
		Scope:      gitlab.String("branches"),
		Status:     gitlab.BuildState(gitlab.Running),
		Ref:        gitlab.String("master"),
		YamlErrors: gitlab.Bool(true),
		Name:       gitlab.String("name"),
		Username:   gitlab.String("username"),
		OrderBy:    gitlab.OrderBy(gitlab.OrderByStatus),
		Sort:       gitlab.String("asc"),
	}

	pipelines, _, err := git.Pipelines.ListProjectPipelines(2743054, opt)
	if err != nil {
		log.Fatal(err)
	}

	for _, pipeline := range pipelines {
		log.Printf("Found pipeline: %v", pipeline)
	}
}
