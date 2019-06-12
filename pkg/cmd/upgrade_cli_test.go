package cmd

import (
	"fmt"
	"github.com/blang/semver"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/stretchr/testify/assert"
	"testing"
)

var sampleBrewInfo = `[  
   {  
      "name":"jx",
      "full_name":"jenkins-x/jx/jx",
      "oldname":null,
      "aliases":[  

      ],
      "versioned_formulae":[  

      ],
      "desc":"A tool to install and interact with Jenkins X on your Kubernetes cluster.",
      "homepage":"https://jenkins-x.github.io/jenkins-x-website/",
      "versions":{  
         "stable":"2.0.181",
         "devel":null,
         "head":null,
         "bottle":false
      },
      "revision":0,
      "version_scheme":0,
      "bottle":{  

      },
      "keg_only":false,
      "bottle_disabled":false,
      "options":[  

      ],
      "build_dependencies":[  

      ],
      "dependencies":[  

      ],
      "recommended_dependencies":[  

      ],
      "optional_dependencies":[  

      ],
      "requirements":[  

      ],
      "conflicts_with":[  

      ],
      "caveats":null,
      "installed":[  
         {  
            "version":"2.0.181",
            "used_options":[  

            ],
            "built_as_bottle":false,
            "poured_from_bottle":false,
            "runtime_dependencies":[  

            ],
            "installed_as_dependency":false,
            "installed_on_request":true
         }
      ],
      "linked_keg":"2.0.181",
      "pinned":false,
      "outdated":false
   }
]`

func TestLatestJxBrewVersion(t *testing.T) {
	o := UpgradeCLIOptions{}
	version, err := o.latestJxBrewVersion(sampleBrewInfo)
	assert.NoError(t, err)
	assert.Equal(t, "2.0.181", version)
}

func TestNeedsUpgrade(t *testing.T) {
	type testData struct {
		current               string
		latest                string
		expectedUpgradeNeeded bool
		expectedMessage       string
	}

	testCases := []testData{
		{
			"1.0.0", "1.0.0", false, "You are already on the latest version of jx 1.0.0\n",
		},
		{
			"1.0.0", "1.0.1", true, "",
		},
		{
			"1.0.0", "0.0.99", true, "",
		},
	}

	o := UpgradeCLIOptions{}
	for _, data := range testCases {
		currentVersion, _ := semver.New(data.current)
		latestVersion, _ := semver.New(data.latest)
		actualMessage := log.CaptureOutput(func() {
			actualUpgradeNeeded := o.needsUpgrade(*currentVersion, *latestVersion)
			assert.Equal(t, data.expectedUpgradeNeeded, actualUpgradeNeeded, fmt.Sprintf("Unexpected upgrade flag for %v", data))
		})
		assert.Equal(t, data.expectedMessage, actualMessage, fmt.Sprintf("Unexpected message for %v", data))
	}
}
