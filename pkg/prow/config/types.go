package config

type Kind string

const (
	Application Kind = "APPLICATION"
	Environment Kind = "ENVIRONMENT"
	Protection  Kind = "PROTECTION"

	ServerlessJenkins = "serverless-jenkins"
	PromotionBuild    = "promotion-build"
)
