package surveyutils_test

import (
	"io/ioutil"
	"path/filepath"
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

func TestBasicTypesValidation(t *testing.T) {
	GenerateValuesAsYaml(t, "basicTypesValidation.test.schema.json",
		func(console *tests.ConsoleWrapper, donec chan struct{}) {
			defer close(donec)
			console.ExpectString("Enter a value for numberValue")
			console.SendLine("abc")
			console.ExpectString("Sorry, your reply was invalid: unable to convert abc to float64")
			console.ExpectString("Enter a value for numberValue")
			console.SendLine("123.1")
			console.ExpectString("Enter a value for integerValue")
			console.SendLine("123.1")
			console.ExpectString("Sorry, your reply was invalid: unable to convert 123.1 to int")
			console.ExpectString("Enter a value for integerValue")
			console.SendLine("123")
			console.ExpectEOF()
		})
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
			console.ExpectString("Sorry, your reply was invalid: nestedObject has less than 1 items, has []")
			console.ExpectString("Enter a value for stringValue")
			console.SendLine("abc")
			console.ExpectString("Enter a value for stringValue1")
			console.SendLine("def")
			console.ExpectEOF()
		})
}

func TestMaxProperties(t *testing.T) {
	GenerateValuesAsYaml(t, "maxProperties.test.schema.json",
		func(console *tests.ConsoleWrapper, donec chan struct{}) {
			defer close(donec)
			// Test boolean type
			console.ExpectString("Enter a value for stringValue")
			console.SendLine("abc")
			console.ExpectString("Enter a value for stringValue1")
			console.SendLine("def")
			console.ExpectString("Sorry, your reply was invalid: nestedObject has more than 1 items, " +
				"has [stringValue stringValue1]")
			console.ExpectString("Enter a value for stringValue")
			console.SendLine("abc")
			console.ExpectString("Enter a value for stringValue1")
			console.SendLine("")
			console.ExpectEOF()
		})
}

func TestDateTime(t *testing.T) {
	GenerateValuesAsYaml(t, "dateTime.test.schema.json",
		func(console *tests.ConsoleWrapper, donec chan struct{}) {
			defer close(donec)
			// Test boolean type
			console.ExpectString("Enter a value for dateTimeValue")
			console.SendLine("abc")
			console.ExpectString("Sorry, your reply was invalid: abc is not a RFC 3339 date-time formatted string, " +
				"it should be like 2006-01-02T15:04:05Z07:00")
			console.ExpectString("Enter a value for dateTimeValue")
			console.SendLine("2006-01-02T15:04:05-07:00")
			console.ExpectEOF()
		})
}

func TestDate(t *testing.T) {
	GenerateValuesAsYaml(t, "date.test.schema.json",
		func(console *tests.ConsoleWrapper, donec chan struct{}) {
			defer close(donec)
			// Test boolean type
			console.ExpectString("Enter a value for dateValue")
			console.SendLine("abc")
			console.ExpectString("Sorry, your reply was invalid: abc is not a RFC 3339 full-date formatted string, " +
				"it should be like 2006-01-02")
			console.ExpectString("Enter a value for dateValue")
			console.SendLine("2006-01-02")
			console.ExpectEOF()
		})
}

func TestTime(t *testing.T) {
	GenerateValuesAsYaml(t, "time.test.schema.json",
		func(console *tests.ConsoleWrapper, donec chan struct{}) {
			defer close(donec)
			// Test boolean type
			console.ExpectString("Enter a value for timeValue")
			console.SendLine("abc")
			console.ExpectString("Sorry, your reply was invalid: abc is not a RFC 3339 full-time formatted string, " +
				"it should be like 15:04:05Z07:00")
			console.ExpectString("Enter a value for timeValue")
			console.SendLine("15:04:05-07:00")
			console.ExpectEOF()
		})
}

func TestEmail(t *testing.T) {
	GenerateValuesAsYaml(t, "email.test.schema.json",
		func(console *tests.ConsoleWrapper, donec chan struct{}) {
			defer close(donec)
			// Test boolean type
			console.ExpectString("Enter a value for emailValue")
			console.SendLine("abc")
			console.ExpectString("Sorry, your reply was invalid: abc is not a RFC 5322 address, " +
				"it should be like Barry Gibb <bg@example.com>")
			console.ExpectString("Enter a value for emailValue")
			console.SendLine("Maurice Gibb <mg@example.com>")
			console.ExpectEOF()
		})
}

func TestIdnEmail(t *testing.T) {
	GenerateValuesAsYaml(t, "idnemail.test.schema.json",
		func(console *tests.ConsoleWrapper, donec chan struct{}) {
			defer close(donec)
			// Test boolean type
			console.ExpectString("Enter a value for emailValue")
			console.SendLine("abc")
			console.ExpectString("Sorry, your reply was invalid: abc is not a RFC 5322 address, " +
				"it should be like Barry Gibb <bg@example.com>")
			console.ExpectString("Enter a value for emailValue")
			console.SendLine("Maurice Gibb <mg@example.com>")
			console.ExpectEOF()
		})
}

func TestHostname(t *testing.T) {
	GenerateValuesAsYaml(t, "hostname.test.schema.json",
		func(console *tests.ConsoleWrapper, donec chan struct{}) {
			defer close(donec)
			// Test boolean type
			console.ExpectString("Enter a value for hostnameValue")
			console.SendLine("*****")
			console.ExpectString("Sorry, your reply was invalid: ***** is not a RFC 1034 hostname, " +
				"it should be like example.com")
			console.ExpectString("Enter a value for hostnameValue")
			console.SendLine("example.com")
			console.ExpectEOF()
		})
}

func TestIdnHostname(t *testing.T) {
	GenerateValuesAsYaml(t, "idnhostname.test.schema.json",
		func(console *tests.ConsoleWrapper, donec chan struct{}) {
			defer close(donec)
			// Test boolean type
			console.ExpectString("Enter a value for hostnameValue")
			console.SendLine("*****")
			console.ExpectString("Sorry, your reply was invalid: ***** is not a RFC 1034 hostname, " +
				"it should be like example.com")
			console.ExpectString("Enter a value for hostnameValue")
			console.SendLine("example.com")
			console.ExpectEOF()
		})
}

func TestIpv4(t *testing.T) {
	GenerateValuesAsYaml(t, "ipv4.test.schema.json",
		func(console *tests.ConsoleWrapper, donec chan struct{}) {
			defer close(donec)
			// Test boolean type
			console.ExpectString("Enter a value for ipv4Value")
			console.SendLine("abc")
			console.ExpectString("Sorry, your reply was invalid: abc is not a RFC 2673 IPv4 Address, " +
				"it should be like 127.0.0.1")
			console.ExpectString("Enter a value for ipv4Value")
			console.SendLine("127.0.0.1")
			console.ExpectEOF()
		})
}

func TestIpv6(t *testing.T) {
	GenerateValuesAsYaml(t, "ipv6.test.schema.json",
		func(console *tests.ConsoleWrapper, donec chan struct{}) {
			defer close(donec)
			// Test boolean type
			console.ExpectString("Enter a value for ipv6Value")
			console.SendLine("abc")
			console.ExpectString("Sorry, your reply was invalid: abc is not a RFC 4291 IPv6 address, " +
				"it should be like ::1")
			console.ExpectString("Enter a value for ipv6Value")
			console.SendLine("::1")
			console.ExpectEOF()
		})
}

func TestUri(t *testing.T) {
	GenerateValuesAsYaml(t, "uri.test.schema.json",
		func(console *tests.ConsoleWrapper, donec chan struct{}) {
			defer close(donec)
			// Test boolean type
			console.ExpectString("Enter a value for uriValue")
			console.SendLine("*****")
			console.ExpectString("Sorry, your reply was invalid: ***** is not a RFC 3986 URI")
			console.ExpectString("Enter a value for uriValue")
			console.SendLine("https://example.com")
			console.ExpectEOF()
		})
}

func TestUriReference(t *testing.T) {
	GenerateValuesAsYaml(t, "uriReference.test.schema.json",
		func(console *tests.ConsoleWrapper, donec chan struct{}) {
			defer close(donec)
			// Test boolean type
			console.ExpectString("Enter a value for uriReferenceValue")
			console.SendLine("http$$://foo")
			console.ExpectString("Sorry, your reply was invalid: http$$://foo is not a RFC 3986 URI reference")
			console.ExpectString("Enter a value for uriReferenceValue")
			console.SendLine("../resource.txt")
			console.ExpectEOF()
		})
}

func TestJSONPointer(t *testing.T) {
	GenerateValuesAsYaml(t, "jsonPointer.test.schema.json",
		func(console *tests.ConsoleWrapper, donec chan struct{}) {
			defer close(donec)
			// Test boolean type
			console.ExpectString("Enter a value for jsonPointerValue")
			console.SendLine("~")
			console.ExpectString("Sorry, your reply was invalid: ~ is not a RFC 6901 JSON pointer")
			console.ExpectString("Enter a value for jsonPointerValue")
			console.SendLine("/abc")
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
	data, err := ioutil.ReadFile(filepath.Join("test_data", schemaName))
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
