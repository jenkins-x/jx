package prow

// Owners keeps the prow OWNERS data
type Owners struct {
	Approvers []string `yaml:"approvers"`
	Reviewers []string `yaml:"reviewers"`
}

// OwnersAliases keept the prow OWNERS_ALIASES data
type OwnersAliases struct {
	Aliases       []string `yaml:"aliases"`
	BestApprovers []string `yaml:"best-approvers"`
	BestReviewers []string `yaml:"best-reviewers"`
}
