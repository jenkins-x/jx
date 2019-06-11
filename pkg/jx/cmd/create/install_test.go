package create_test

import (
	"errors"
	"github.com/jenkins-x/jx/pkg/jx/cmd/create"
	"github.com/jenkins-x/jx/pkg/jx/cmd/initcmd"
	"os"
	"path"
	"testing"

	"fmt"

	configio "github.com/jenkins-x/jx/pkg/io"
	"github.com/jenkins-x/jx/pkg/jx/cmd/opts"
	"github.com/jenkins-x/jx/pkg/util"

	//. "github.com/petergtz/pegomock"
	"github.com/stretchr/testify/assert"
)

func TestInstall(t *testing.T) {
	t.Parallel()
	testDir := path.Join("test_data", "install_jenkins_x_versions")
	_, err := os.Stat(testDir)
	assert.NoError(t, err)

	configStore := configio.NewFileStore()
	version, err := create.LoadVersionFromCloudEnvironmentsDir(testDir, configStore)
	assert.NoError(t, err)

	assert.Equal(t, "0.0.3321", version, "For stable version in dir %s", testDir)
}

func TestGenerateProwSecret(t *testing.T) {
	fmt.Println(util.RandStringBytesMaskImprSrc(41))
}

func TestGetSafeUsername(t *testing.T) {
	t.Parallel()
	username := `Your active configuration is: [cloudshell-16392]
tutorial@bamboo-depth-206411.iam.gserviceaccount.com`
	assert.Equal(t, opts.GetSafeUsername(username), "tutorial@bamboo-depth-206411.iam.gserviceaccount.com")

	username = `tutorial@bamboo-depth-206411.iam.gserviceaccount.com`
	assert.Equal(t, opts.GetSafeUsername(username), "tutorial@bamboo-depth-206411.iam.gserviceaccount.com")
}

func TestCheckFlags(t *testing.T) {

	var tests = []struct {
		name           string
		in             *create.InstallFlags
		nextGeneration bool
		tekton         bool
		prow           bool
		staticJenkins  bool
		knativeBuild   bool
		err            error
	}{
		{
			name:           "default",
			in:             &create.InstallFlags{},
			nextGeneration: false,
			tekton:         false,
			prow:           false,
			staticJenkins:  true,
			knativeBuild:   false,
			err:            nil,
		},
		{
			name: "next_generation",
			in: &create.InstallFlags{
				NextGeneration: true,
			},
			nextGeneration: true,
			tekton:         true,
			prow:           true,
			staticJenkins:  false,
			knativeBuild:   false,
			err:            nil,
		},
		{
			name: "prow",
			in: &create.InstallFlags{
				Prow: true,
			},
			nextGeneration: false,
			tekton:         true,
			prow:           true,
			staticJenkins:  false,
			knativeBuild:   false,
			err:            nil,
		},
		{
			name: "prow_and_knative",
			in: &create.InstallFlags{
				Prow:         true,
				KnativeBuild: true,
			},
			nextGeneration: false,
			tekton:         false,
			prow:           true,
			staticJenkins:  false,
			knativeBuild:   true,
			err:            nil,
		},
		{
			name: "next_generation_and_static_jenkins",
			in: &create.InstallFlags{
				NextGeneration: true,
				StaticJenkins:  true,
			},
			err: errors.New("Incompatible options '--ng' and '--static-jenkins'. Please pick only one of them. We recommend --ng as --static-jenkins is deprecated"),
		},
		{
			name: "tekton_and_static_jenkins",
			in: &create.InstallFlags{
				Tekton:        true,
				StaticJenkins: true,
			},
			err: errors.New("Incompatible options '--tekton' and '--static-jenkins'. Please pick only one of them. We recommend --tekton as --static-jenkins is deprecated"),
		},
		{
			name: "tekton_and_knative",
			in: &create.InstallFlags{
				Tekton:       true,
				KnativeBuild: true,
			},
			err: errors.New("Incompatible options '--knative-build' and '--tekton'. Please pick only one of them. We recommend --tekton as --knative-build is deprecated"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := create.InstallOptions{
				CommonOptions: &opts.CommonOptions{
					BatchMode: true,
				},
				Flags:       *tt.in,
				InitOptions: initcmd.InitOptions{},
			}

			err := opts.CheckFlags()
			if tt.err != nil {
				assert.Equal(t, tt.err, err)
			} else {
				assert.NoError(t, err)

				assert.Equal(t, tt.nextGeneration, opts.Flags.NextGeneration, "NextGeneration flag is not as expected")
				assert.Equal(t, tt.tekton, opts.Flags.Tekton, "Tekton flag is not as expected")
				assert.Equal(t, tt.prow, opts.Flags.Prow, "Prow flag is not as expected")
				assert.Equal(t, tt.staticJenkins, opts.Flags.StaticJenkins, "StaticJenkins flag is not as expected")
				assert.Equal(t, tt.knativeBuild, opts.Flags.KnativeBuild, "KnativeBuild flag is not as expected")
			}
		})
	}
}

func TestInstallRun(t *testing.T) {
	// Create mocks...
	//factory := cmd_mocks.NewMockFactory()
	//kubernetesInterface := kube_mocks.NewSimpleClientset()
	//// Override CreateKubeClient to return mock Kubernetes interface
	//When(factory.CreateKubeClient()).ThenReturn(kubernetesInterface, "jx-testing", nil)

	//options := cmd.CreateInstallOptions(factory, os.Stdin, os.Stdout, os.Stderr)

	//err := options.Run()

	//assert.NoError(t, err, "Should not error")
}

func TestVerifyDomainName(t *testing.T) {
	t.Parallel()
	invalidErr := "domain name %s contains invalid characters"
	lengthErr := "domain name %s has fewer than 3 or greater than 63 characters"

	domain := "wine.com"
	assert.Equal(t, create.ValidateDomainName(domain), nil)
	domain = "more-wine.com"
	assert.Equal(t, create.ValidateDomainName(domain), nil)
	domain = "wine-and-cheese.com"
	assert.Equal(t, create.ValidateDomainName(domain), nil)
	domain = "wine-and-cheese.tasting.com"
	assert.Equal(t, create.ValidateDomainName(domain), nil)
	domain = "wine123.com"
	assert.Equal(t, create.ValidateDomainName(domain), nil)
	domain = "wine.cheese.com"
	assert.Equal(t, create.ValidateDomainName(domain), nil)
	domain = "win_e.com"
	assert.Equal(t, create.ValidateDomainName(domain), nil)

	domain = "win?e.com"
	assert.EqualError(t, create.ValidateDomainName(domain), fmt.Sprintf(invalidErr, domain))
	domain = "win%e.com"
	assert.EqualError(t, create.ValidateDomainName(domain), fmt.Sprintf(invalidErr, domain))
	domain = "om"

	assert.EqualError(t, create.ValidateDomainName(domain), fmt.Sprintf(lengthErr, domain))
	domain = "some.really.long.domain.that.should.be.longer.than.the.maximum.63.characters.com"
	assert.EqualError(t, create.ValidateDomainName(domain), fmt.Sprintf(lengthErr, domain))
}

func TestStripTrailingSlash(t *testing.T) {
	t.Parallel()

	url := "http://some.url.com/"
	assert.Equal(t, create.StripTrailingSlash(url), "http://some.url.com")

	url = "http://some.other.url.com"
	assert.Equal(t, create.StripTrailingSlash(url), "http://some.other.url.com")
}
