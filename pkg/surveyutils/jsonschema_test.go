package surveyutils_test

import (
	"io/ioutil"
	"testing"

	"github.com/Netflix/go-expect"

	"gopkg.in/AlecAivazis/survey.v1/core"

	"github.com/ghodss/yaml"

	"github.com/jenkins-x/jx/pkg/tests"

	"github.com/stretchr/testify/assert"

	jenkinsv1 "github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/surveyutils"
)

// TODO Figure out how to test selects

type GeneratedSecret struct {
	Name  string
	Key   string
	Value string
}

func init() {
	// disable color output for all prompts to simplify testing
	core.DisableColor = true
}

func TestObjectType(t *testing.T) {
	assert.Equal(t, `nestedObject:
  anotherNestedObject:
    name: cheese
`, GenerateValuesAsYaml(t, "objectType.test.schema.json",
		func(console *tests.ConsoleWrapper, donec chan struct{}) {
			defer close(donec)
			console.ExpectString("Enter a value for name")
			console.SendLine("cheese")
			console.ExpectEOF()
		}))
}

func TestDescriptionAndTitle(t *testing.T) {
	assert.Equal(t, `address: '?'
country: UK
name: Pete
`, GenerateValuesAsYaml(t, "descriptionAndTitle.test.schema.json",
		func(console *tests.ConsoleWrapper, donec chan struct{}) {
			defer close(donec)
			// Test explicit question
			console.ExpectString("What is your name?")
			console.SendLine("?")
			// Test explicit description
			console.ExpectString("Enter your name")
			console.SendLine("Pete")
			// Test no description
			console.ExpectString("What is your address?")
			console.SendLine("?")
			// Test no title
			console.ExpectString("Enter a value for country")
			console.SendLine("UK")
			console.ExpectEOF()
		}))
}

func TestDefaultValues(t *testing.T) {
	assert.Equal(t, `booleanValue: false
integerValue: 123
numberValue: 123.4
stringValue: UK
`, GenerateValuesAsYaml(t, "defaultValues.test.schema.json",
		func(console *tests.ConsoleWrapper, donec chan struct{}) {
			defer close(donec)
			// Test default value
			console.ExpectString("Enter a value for stringValue (UK)")
			console.SendLine("")
			console.ExpectString("Enter a value for booleanValue (y/N)")
			console.SendLine("")
			console.ExpectString("Enter a value for numberValue (123.4)")
			console.SendLine("")
			console.ExpectString("Enter a value for integerValue (123)")
			console.SendLine("")
			console.ExpectEOF()
		}))
}

func TestConstValues(t *testing.T) {
	assert.Equal(t, `booleanValue: false
integerValue: 123
numberValue: 123.4
stringValue: UK
`, GenerateValuesAsYaml(t, "constValues.test.schema.json",
		func(console *tests.ConsoleWrapper, donec chan struct{}) {
			defer close(donec)
			// Test default value
			console.ExpectString("Do you want to set stringValue to UK (Y/n)")
			console.SendLine("")
			console.ExpectString("Do you want to set booleanValue to false (Y/n)")
			console.SendLine("")
			console.ExpectString("Do you want to set numberValue to 123.4 (Y/n)")
			console.SendLine("")
			console.ExpectString("Do you want to set integerValue to 123")
			console.SendLine("")
			console.ExpectEOF()
		}))
}

func TestBasicTypes(t *testing.T) {
	assert.Equal(t, `booleanValue: true
integerValue: 123
nullValue: null
numberValue: 123.4
stringValue: hello
`, GenerateValuesAsYaml(t, "basicTypes.test.schema.json",
		func(console *tests.ConsoleWrapper, donec chan struct{}) {
			defer close(donec)
			// Test boolean type
			console.ExpectString("Enter a value for booleanValue (y/N)")
			console.SendLine("Y")
			console.ExpectString("Enter a value for numberValue")
			console.SendLine("123.4")
			console.ExpectString("Enter a value for stringValue")
			console.SendLine("hello")
			console.ExpectString("Enter a value for integerValue")
			console.SendLine("123")
			console.ExpectEOF()
		}))
}

func GenerateValuesAsYaml(t *testing.T, schemaName string, answerQuestions func(console *tests.
	ConsoleWrapper, donec chan struct{})) string {
	tests.SkipForWindows(t, "go-expect does not work on windows")
	secrets := make([]*GeneratedSecret, 0)
	options := surveyutils.JSONSchemaOptions{
		CreateSecret: func(name string, key string, value string) (*jenkinsv1.ResourceReference, error) {
			secrets = append(secrets, &GeneratedSecret{
				Name:  name,
				Value: value,
				Key:   key,
			})
			return &jenkinsv1.ResourceReference{
				Name: name,
				Kind: "Secret",
			}, nil
		},
	}
	data, err := ioutil.ReadFile(schemaName)
	assert.NoError(t, err)
	console := tests.NewTerminal(t)
	// Test interactive IO
	donec := make(chan struct{})
	go answerQuestions(console, donec)
	assert.NoError(t, err)
	result, err := options.GenerateValues(data, make([]string, 0), console.In, console.Out, console.Err)
	assert.NoError(t, err)
	err = console.Close()
	<-donec
	assert.NoError(t, err)
	yaml, err := yaml.JSONToYAML(result)
	t.Logf(expect.StripTrailingEmptyLines(console.CurrentState()))
	assert.NoError(t, err)
	return string(yaml)
}
