package sourcerepository

import "github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"

// Applicationer is responsible for storing information about Applications (aka Projects)
//go:generate pegomock generate github.com/jenkins-x/jx/pkg/sourcerepository SourceRepoer -o mocks/sourcerepository.go --generate-matchers
// FIXME - note. At the moment, applications are only referred to by their name (ie, not the organisation) meaning you
// can't import both github.com/org1/myawesomeapp and github.com/org2/myawesomeapp.
type SourceRepoer interface {
	//CreateSourceRepository creates an application. If an application already exists, it will return an error
	CreateSourceRepository(name, organisation, providerUrl string) error

	// DeleteSourceRepository deletes an application
	DeleteSourceRepository(name string) error

	// GetSourceRepository gets an application, if it exists and returns an error otherwise
	GetSourceRepository(name string) (*v1.SourceRepository, error)
}
