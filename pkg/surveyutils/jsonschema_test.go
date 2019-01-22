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

// TODO Figure out how to test selects (affects arrays, enums, validation keywords for arrays)

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

func TestMultipleOf(t *testing.T) {
	GenerateValuesAsYaml(t, "multipleOf.test.schema.json",
		func(console *tests.ConsoleWrapper, donec chan struct{}) {
			defer close(donec)
			// Test boolean type
			console.ExpectString("Enter a value for numberValue")
			console.SendLine("11.1")
			console.ExpectString("Sorry, your reply was invalid: 11.1 cannot be divided by 10")
			console.ExpectString("Enter a value for numberValue")
			console.SendLine("10")
			console.ExpectString("Enter a value for integerValue")
			console.SendLine("12")
			console.ExpectString("Sorry, your reply was invalid: 12 cannot be divided by 20")
			console.ExpectString("Enter a value for integerValue")
			console.SendLine("20")
			console.ExpectEOF()
		})
}

func TestMaximum(t *testing.T) {
	GenerateValuesAsYaml(t, "maximum.test.schema.json",
		func(console *tests.ConsoleWrapper, donec chan struct{}) {
			defer close(donec)
			// Test boolean type
			console.ExpectString("Enter a value for numberValue")
			console.SendLine("11.1")
			console.ExpectString("Sorry, your reply was invalid: 11.1 is not less than or equal to 10.1")
			console.ExpectString("Enter a value for numberValue")
			console.SendLine("1")
			console.ExpectString("Enter a value for integerValue")
			console.SendLine("21")
			console.ExpectString("Sorry, your reply was invalid: 21 is not less than or equal to 20")
			console.ExpectString("Enter a value for integerValue")
			console.SendLine("2")
			console.ExpectEOF()
		})
}

func TestExclusiveMaximum(t *testing.T) {
	GenerateValuesAsYaml(t, "exclusiveMaximum.test.schema.json",
		func(console *tests.ConsoleWrapper, donec chan struct{}) {
			defer close(donec)
			// Test boolean type
			console.ExpectString("Enter a value for numberValue")
			console.SendLine("10.1")
			console.ExpectString("Sorry, your reply was invalid: 10.1 is not less than 10.1")
			console.ExpectString("Enter a value for numberValue")
			console.SendLine("1")
			console.ExpectString("Enter a value for integerValue")
			console.SendLine("20")
			console.ExpectString("Sorry, your reply was invalid: 20 is not less than 20")
			console.ExpectString("Enter a value for integerValue")
			console.SendLine("2")
			console.ExpectEOF()
		})
}

func TestMinimum(t *testing.T) {
	GenerateValuesAsYaml(t, "minimum.test.schema.json",
		func(console *tests.ConsoleWrapper, donec chan struct{}) {
			defer close(donec)
			// Test boolean type
			console.ExpectString("Enter a value for numberValue")
			console.SendLine("9.1")
			console.ExpectString("Sorry, your reply was invalid: 9.1 is not greater than or equal to 10.1")
			console.ExpectString("Enter a value for numberValue")
			console.SendLine("11")
			console.ExpectString("Enter a value for integerValue")
			console.SendLine("19")
			console.ExpectString("Sorry, your reply was invalid: 19 is not greater than or equal to 20")
			console.ExpectString("Enter a value for integerValue")
			console.SendLine("21")
			console.ExpectEOF()
		})
}

func TestExclusiveMinimum(t *testing.T) {
	GenerateValuesAsYaml(t, "exclusiveMinimum.test.schema.json",
		func(console *tests.ConsoleWrapper, donec chan struct{}) {
			defer close(donec)
			// Test boolean type
			console.ExpectString("Enter a value for numberValue")
			console.SendLine("10.1")
			console.ExpectString("Sorry, your reply was invalid: 10.1 is not greater than 10.1")
			console.ExpectString("Enter a value for numberValue")
			console.SendLine("11")
			console.ExpectString("Enter a value for integerValue")
			console.SendLine("20")
			console.ExpectString("Sorry, your reply was invalid: 20 is not greater than 20")
			console.ExpectString("Enter a value for integerValue")
			console.SendLine("21")
			console.ExpectEOF()
		})
}

func TestMaxLength(t *testing.T) {
	GenerateValuesAsYaml(t, "maxLength.test.schema.json",
		func(console *tests.ConsoleWrapper, donec chan struct{}) {
			defer close(donec)
			// Test boolean type
			console.ExpectString("Enter a value for stringValue")
			console.SendLine("iamlongerthan10")
			console.ExpectString("Sorry, your reply was invalid: value is too long. Max length is 10")
			console.ExpectString("Enter a value for stringValue")
			console.SendLine("short")
			console.ExpectEOF()
		})
}

func TestMinLength(t *testing.T) {
	GenerateValuesAsYaml(t, "minLength.test.schema.json",
		func(console *tests.ConsoleWrapper, donec chan struct{}) {
			defer close(donec)
			// Test boolean type
			console.ExpectString("Enter a value for stringValue")
			console.SendLine("short")
			console.ExpectString("Sorry, your reply was invalid: value is too short. Min length is 10")
			console.ExpectString("Enter a value for stringValue")
			console.SendLine("iamlongerthan10")
			console.ExpectEOF()
		})
}

func TestPattern(t *testing.T) {
	GenerateValuesAsYaml(t, "pattern.test.schema.json",
		func(console *tests.ConsoleWrapper, donec chan struct{}) {
			defer close(donec)
			// Test boolean type
			console.ExpectString("Enter a value for stringValue")
			console.SendLine("HELLO")
			console.ExpectString("Sorry, your reply was invalid: HELLO does not match [0-9]")
			console.ExpectString("Enter a value for stringValue")
			console.SendLine("123")
			console.ExpectEOF()
		})
}

func TestRequired(t *testing.T) {
	GenerateValuesAsYaml(t, "required.test.schema.json",
		func(console *tests.ConsoleWrapper, donec chan struct{}) {
			defer close(donec)
			// Test boolean type
			console.ExpectString("Enter a value for stringValue")
			console.SendLine("")
			console.ExpectString("Sorry, your reply was invalid: Value is required")
			console.ExpectString("Enter a value for stringValue")
			console.SendLine("Hello")
			console.ExpectEOF()
		})
}

func TestMinProperties(t *testing.T) {
	GenerateValuesAsYaml(t, "minProperties.test.schema.json",
		func(console *tests.ConsoleWrapper, donec chan struct{}) {
			defer close(donec)
			// Test boolean type
			console.ExpectString("Enter a value for stringValue")
			console.SendLine("")
			console.ExpectString("Enter a value for stringValue1")
			console.SendLine("")
			console.ExpectString("nestedObject has less than 1 items")
			console.ExpectString("Enter a value for stringValue")
			console.SendLine("abc")
			console.ExpectString("Enter a value for stringValue1")
			console.SendLine("def")
			console.ExpectEOF()
		})
}


func GenerateValuesAsYaml(t *testing.T, schemaName string, answerQuestions func(console *tests.
	ConsoleWrapper, donec chan struct{})) string {
	tests.SkipForWindows(t, "go-expect does not work on windows")
	t.Parallel()
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
