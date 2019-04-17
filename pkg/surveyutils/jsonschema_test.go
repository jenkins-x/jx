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
	values, _, err := GenerateValuesAsYaml(t, "objectType.test.schema.json", make(map[string]interface{}), false, false,
		false, false,
		func(console *tests.ConsoleWrapper, donec chan struct{}) {
			defer close(donec)
			console.ExpectString("Enter a value for name")
			console.SendLine("cheese")
			console.ExpectEOF()
		})
	assert.Equal(t, `nestedObject:
  anotherNestedObject:
    name: cheese
`, values)
	assert.NoError(t, err)
}

func TestDescriptionAndTitle(t *testing.T) {
	values, _, err := GenerateValuesAsYaml(t, "descriptionAndTitle.test.schema.json", make(map[string]interface{}),
		false,
		false, false, false,
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
		})
	assert.NoError(t, err)
	assert.Equal(t, `address: '?'
country: UK
name: Pete
`, values)
}

func TestAutoAcceptDefaultValues(t *testing.T) {
	values, _, err := GenerateValuesAsYaml(t, "autoAcceptDefaultValues.test.schema.json", make(map[string]interface{}),
		false, false,
		true, false,
		func(console *tests.ConsoleWrapper, donec chan struct{}) {
			defer close(donec)
			// Test explicit question
			//console.ExpectString("What is your name? John Smith [Automatically accepted default value]")
			//console.ExpectEOF()
			// TODO Fix the console test
		})
	assert.Equal(t, `name: John Smith
`, values)
	assert.NoError(t, err)
}

func TestAcceptExisting(t *testing.T) {
	t.SkipNow()
	// TODO Fix failing test
	values, _, err := GenerateValuesAsYaml(t, "acceptExisting.test.schema.json", map[string]interface{}{
		"name": "John Smith",
	},
		false, false,
		false, false,
		func(console *tests.ConsoleWrapper, donec chan struct{}) {
			defer close(donec)
			// Test explicit question
			console.ExpectString("What is your name? John Smith [Automatically accepted existing value]")
			console.ExpectEOF()
		})
	assert.Equal(t, `name: John Smith
`, values)
	assert.NoError(t, err)
}

func TestAskExisting(t *testing.T) {
	values, _, err := GenerateValuesAsYaml(t, "askExisting.test.schema.json", map[string]interface{}{
		"name": "John Smith",
	},
		true,
		false, false, false,
		func(console *tests.ConsoleWrapper, donec chan struct{}) {
			defer close(donec)
			// Test explicit question
			console.ExpectString("What is your name? [? for help] (John Smith)")
			console.SendLine("")
			console.ExpectEOF()
		})
	assert.NoError(t, err)
	assert.Equal(t, `name: John Smith
`, values)
}

func TestNoAskAndAutoAcceptDefaultsWithExisting(t *testing.T) {
	// TODO Fix the flacky console tests and reenable this test again
	t.Skip()
	values, _, err := GenerateValuesAsYaml(t, "noAskAndAutoAcceptDefaultsWithExisting.test.schema.json",
		map[string]interface{}{
			"name":    "John Smith",
			"country": "UK",
		},
		false,
		true, true, false,
		func(console *tests.ConsoleWrapper, donec chan struct{}) {
			defer close(donec)
			// Test explicit question
			console.ExpectString("What is your name? John Smith [Automatically accepted existing value]")
			console.ExpectString("Enter a value for country UK [Automatically accepted default value]")
			console.ExpectEOF()
		})
	assert.NoError(t, err)
	assert.Equal(t, `country: UK
name: John Smith
`, values)
}

func TestIgnoreMissingValues(t *testing.T) {
	values, _, err := GenerateValuesAsYaml(t, "ignoreMissingValues.test.schema.json", make(map[string]interface{}),
		false,
		true,
		false, true,
		func(console *tests.ConsoleWrapper, donec chan struct{}) {
			defer close(donec)
			console.ExpectEOF()
		})
	assert.NoError(t, err)
	assert.Equal(t, `{}
`, values)
}

func TestErrorMissingValues(t *testing.T) {
	_, _, err := GenerateValuesAsYaml(t, "ignoreMissingValues.test.schema.json", make(map[string]interface{}),
		false,
		true,
		false, false,
		func(console *tests.ConsoleWrapper, donec chan struct{}) {
			defer close(donec)
			console.ExpectEOF()
		})
	assert.Error(t, err)
}

func TestDefaultValues(t *testing.T) {
	values, _, err := GenerateValuesAsYaml(t, "defaultValues.test.schema.json", make(map[string]interface{}), false,
		false, false, false,
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
		})
	assert.NoError(t, err)
	assert.Equal(t, `booleanValue: false
integerValue: 123
numberValue: 123.4
stringValue: UK
`, values)
}

func TestConstValues(t *testing.T) {
	values, _, err := GenerateValuesAsYaml(t, "constValues.test.schema.json", make(map[string]interface{}), false,
		false,
		false, false,
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
		})
	assert.NoError(t, err)
	assert.Equal(t, `booleanValue: false
integerValue: 123
numberValue: 123.4
stringValue: UK
`, values)
}

func TestBasicTypesValidation(t *testing.T) {
	_, _, err := GenerateValuesAsYaml(t, "basicTypesValidation.test.schema.json", make(map[string]interface{}), false,
		false,
		false, false,
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
	assert.NoError(t, err)
}

func TestBasicTypes(t *testing.T) {
	values, _, err := GenerateValuesAsYaml(t, "basicTypes.test.schema.json", make(map[string]interface{}), false, false,
		false, false,
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
		})
	assert.Equal(t, `booleanValue: true
integerValue: 123
nullValue: null
numberValue: 123.4
stringValue: hello
`, values)
	assert.NoError(t, err)
}

func TestMultipleOf(t *testing.T) {
	_, _, err := GenerateValuesAsYaml(t, "multipleOf.test.schema.json", make(map[string]interface{}), false, false, false, false,
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
	assert.NoError(t, err)
}

func TestMaximum(t *testing.T) {
	_, _, err := GenerateValuesAsYaml(t, "maximum.test.schema.json", make(map[string]interface{}), false, false, false, false,
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
	assert.NoError(t, err)
}

func TestExclusiveMaximum(t *testing.T) {
	_, _, err := GenerateValuesAsYaml(t, "exclusiveMaximum.test.schema.json", make(map[string]interface{}), false, false, false,
		false,
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
	assert.NoError(t, err)
}

func TestMinimum(t *testing.T) {
	_, _, err := GenerateValuesAsYaml(t, "minimum.test.schema.json", make(map[string]interface{}), false, false, false, false,
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
	assert.NoError(t, err)
}

func TestExclusiveMinimum(t *testing.T) {
	_, _, err := GenerateValuesAsYaml(t, "exclusiveMinimum.test.schema.json", make(map[string]interface{}), false, false, false,
		false,
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
	assert.NoError(t, err)
}

func TestMaxLength(t *testing.T) {
	_, _, err := GenerateValuesAsYaml(t, "maxLength.test.schema.json", make(map[string]interface{}), false, false, false, false,
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
	assert.NoError(t, err)
}

func TestMinLength(t *testing.T) {
	_, _, err := GenerateValuesAsYaml(t, "minLength.test.schema.json", make(map[string]interface{}), false, false, false, false,
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
	assert.NoError(t, err)
}

func TestPattern(t *testing.T) {
	_, _, err := GenerateValuesAsYaml(t, "pattern.test.schema.json", make(map[string]interface{}), false, false, false, false,
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
	assert.NoError(t, err)
}

func TestRequired(t *testing.T) {
	_, _, err := GenerateValuesAsYaml(t, "required.test.schema.json", make(map[string]interface{}), false, false, false, false,
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
	assert.NoError(t, err)
}

func TestIfThen(t *testing.T) {
	values, _, err := GenerateValuesAsYaml(t, "ifThenElse.test.schema.json", make(map[string]interface{}), false, false, false, false,
		func(console *tests.ConsoleWrapper, donec chan struct{}) {
			defer close(donec)
			console.ExpectString("Enter a value for enablePersistentStorage")
			console.SendLine("Y")
			console.ExpectString("Enter a value for databaseConnectionUrl")
			console.SendLine("abc")
			console.ExpectString("Enter a value for databaseUsername")
			console.SendLine("wensleydale")
			console.ExpectString("Enter a value for databasePassword")
			console.SendLine("cranberries")
			console.ExpectString(" ***********")
			console.ExpectEOF()
		})
	assert.NoError(t, err)
	assert.Equal(t, `databaseConnectionUrl: abc
databasePassword:
  kind: Secret
  name: databasepassword-secret
databaseUsername: wensleydale
enablePersistentStorage: true
`, values)
}

func TestIfElse(t *testing.T) {
	values, _, err := GenerateValuesAsYaml(t, "ifThenElse.test.schema.json", make(map[string]interface{}), false, false, false, false,
		func(console *tests.ConsoleWrapper, donec chan struct{}) {
			defer close(donec)
			console.ExpectString("Enter a value for enablePersistentStorage")
			console.SendLine("N")
			console.ExpectString("Enter a value for enableInMemoryDB")
			console.SendLine("N")
			console.ExpectEOF()
		})
	assert.NoError(t, err)
	assert.Equal(t, `enableInMemoryDB: false
enablePersistentStorage: false
`, values)
}

func TestIfElseNested(t *testing.T) {
	values, _, err := GenerateValuesAsYaml(t, "ifThenElseNested.test.schema.json", make(map[string]interface{}), false, false, false, false,
		func(console *tests.ConsoleWrapper, donec chan struct{}) {
			defer close(donec)
			console.ExpectString("Enter a value for enablePersistentStorage")
			console.SendLine("N")
			console.ExpectString("Enter a value for enableInMemoryDB")
			console.SendLine("Y")
			console.ExpectString("Enter a value for nestedString")
			console.SendLine("Test")
			console.ExpectEOF()
		})
	assert.NoError(t, err)
	assert.Equal(t, `nestedObject:
  enableInMemoryDB: true
  enablePersistentStorage: false
  nestedString: Test
`, values)
}

func TestIfElseWithDefaults(t *testing.T) {
	values, _, err := GenerateValuesAsYaml(t, "ifThenElse.test.schema.json", make(map[string]interface{}), false, false, true, false,
		func(console *tests.ConsoleWrapper, donec chan struct{}) {
			defer close(donec)
			console.ExpectString("Enter a value for enablePersistentStorage")
			console.SendLine("N")
			console.ExpectEOF()
		})
	assert.NoError(t, err)
	assert.Equal(t, `enableInMemoryDB: true
enablePersistentStorage: false
`, values)
}

func TestAllOf(t *testing.T) {
	values, _, err := GenerateValuesAsYaml(t, "AllOfIf.test.schema.json", make(map[string]interface{}), false, false, false, false,
		func(console *tests.ConsoleWrapper, donec chan struct{}) {
			defer close(donec)
			console.ExpectString("Enter a value for enablePersistentStorage")
			console.SendLine("Y")
			console.ExpectString("Enter a value for databaseConnectionUrl")
			console.SendLine("abc")
			console.ExpectString("Enter a value for databaseUsername")
			console.SendLine("wensleydale")
			console.ExpectString("Enter a value for databasePassword")
			console.SendLine("cranberries")
			console.ExpectString(" ***********")
			console.ExpectString("Enter a value for enableCheese")
			console.SendLine("Y")
			console.ExpectString("Enter a value for cheeseType")
			console.SendLine("Stilton")
			console.ExpectEOF()
		})
	assert.NoError(t, err)
	assert.Equal(t, `cheeseType: Stilton
databaseConnectionUrl: abc
databasePassword:
  kind: Secret
  name: databasepassword-secret
databaseUsername: wensleydale
enableCheese: true
enablePersistentStorage: true
`, values)
}

func TestAllOfThen(t *testing.T) {
	values, _, err := GenerateValuesAsYaml(t, "AllOfIf.test.schema.json", make(map[string]interface{}), false, false, false, false,
		func(console *tests.ConsoleWrapper, donec chan struct{}) {
			defer close(donec)
			console.ExpectString("Enter a value for enablePersistentStorage")
			console.SendLine("Y")
			console.ExpectString("Enter a value for databaseConnectionUrl")
			console.SendLine("abc")
			console.ExpectString("Enter a value for databaseUsername")
			console.SendLine("wensleydale")
			console.ExpectString("Enter a value for databasePassword")
			console.SendLine("cranberries")
			console.ExpectString(" ***********")
			console.ExpectString("Enter a value for enableCheese")
			console.SendLine("N")
			console.ExpectString("Enter a value for iDontLikeCheese")
			console.SendLine("Y")
			console.ExpectEOF()
		})
	assert.NoError(t, err)
	assert.Equal(t, `databaseConnectionUrl: abc
databasePassword:
  kind: Secret
  name: databasepassword-secret
databaseUsername: wensleydale
enableCheese: false
enablePersistentStorage: true
iDontLikeCheese: true
`, values)
}

func TestMinProperties(t *testing.T) {
	_, _, err := GenerateValuesAsYaml(t, "minProperties.test.schema.json", make(map[string]interface{}), false, false, false, false,
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
	assert.NoError(t, err)
}

func TestMaxProperties(t *testing.T) {
	_, _, err := GenerateValuesAsYaml(t, "maxProperties.test.schema.json", make(map[string]interface{}), false, false, false, false,
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
	assert.NoError(t, err)
}

func TestDateTime(t *testing.T) {
	_, _, err := GenerateValuesAsYaml(t, "dateTime.test.schema.json", make(map[string]interface{}), false, false, false, false,
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
	assert.NoError(t, err)
}

func TestDate(t *testing.T) {
	_, _, err := GenerateValuesAsYaml(t, "date.test.schema.json", make(map[string]interface{}), false, false, false, false,
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
	assert.NoError(t, err)
}

func TestTime(t *testing.T) {
	_, _, err := GenerateValuesAsYaml(t, "time.test.schema.json", make(map[string]interface{}), false, false, false, false,
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
	assert.NoError(t, err)
}

func TestPassword(t *testing.T) {
	values, secrets, err := GenerateValuesAsYaml(t, "password.test.schema.json", make(map[string]interface{}), false,
		false, false, false,
		func(console *tests.ConsoleWrapper, donec chan struct{}) {
			defer close(donec)
			// Test boolean type
			console.ExpectString("Enter a value for passwordValue")
			console.SendLine("abc")
			console.ExpectEOF()
		})
	assert.Equal(t, `passwordValue:
  kind: Secret
  name: passwordvalue-secret
`, values)
	assert.Contains(t, secrets, &GeneratedSecret{
		Name:  "passwordvalue-secret",
		Value: "abc",
		Key:   "password",
	})
	assert.NoError(t, err)
}

func TestToken(t *testing.T) {
	values, secrets, err := GenerateValuesAsYaml(t, "token.test.schema.json", make(map[string]interface{}), false,
		false,
		false, false,
		func(console *tests.ConsoleWrapper, donec chan struct{}) {
			defer close(donec)
			// Test boolean type
			console.ExpectString("Enter a value for tokenValue")
			console.SendLine("abc")
			console.ExpectEOF()
		})
	assert.Equal(t, `tokenValue:
  kind: Secret
  name: tokenvalue-secret
`, values)
	assert.Contains(t, secrets, &GeneratedSecret{
		Name:  "tokenvalue-secret",
		Value: "abc",
		Key:   "token",
	})
	assert.NoError(t, err)
}

func TestEmail(t *testing.T) {
	_, _, err := GenerateValuesAsYaml(t, "email.test.schema.json", make(map[string]interface{}), false, false, false,
		false,
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
	assert.NoError(t, err)
}

func TestIdnEmail(t *testing.T) {
	_, _, err := GenerateValuesAsYaml(t, "idnemail.test.schema.json", make(map[string]interface{}), false, false, false, false,
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
	assert.NoError(t, err)
}

func TestHostname(t *testing.T) {
	_, _, err := GenerateValuesAsYaml(t, "hostname.test.schema.json", make(map[string]interface{}), false, false, false, false,
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
	assert.NoError(t, err)
}

func TestIdnHostname(t *testing.T) {
	_, _, err := GenerateValuesAsYaml(t, "idnhostname.test.schema.json", make(map[string]interface{}), false, false, false, false,
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
	assert.NoError(t, err)
}

func TestIpv4(t *testing.T) {
	_, _, err := GenerateValuesAsYaml(t, "ipv4.test.schema.json", make(map[string]interface{}), false, false, false, false,
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
	assert.NoError(t, err)
}

func TestIpv6(t *testing.T) {
	_, _, err := GenerateValuesAsYaml(t, "ipv6.test.schema.json", make(map[string]interface{}), false, false, false, false,
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
	assert.NoError(t, err)
}

func TestUri(t *testing.T) {
	_, _, err := GenerateValuesAsYaml(t, "uri.test.schema.json", make(map[string]interface{}), false, false, false, false,
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
	assert.NoError(t, err)
}

func TestUriReference(t *testing.T) {
	_, _, err := GenerateValuesAsYaml(t, "uriReference.test.schema.json", make(map[string]interface{}), false, false, false, false,
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
	assert.NoError(t, err)
}

func TestJSONPointer(t *testing.T) {
	_, _, err := GenerateValuesAsYaml(t, "jsonPointer.test.schema.json", make(map[string]interface{}), false, false, false, false,
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
	assert.NoError(t, err)

}

func GenerateValuesAsYaml(t *testing.T, schemaName string, existingValues map[string]interface{},
	askExisting bool, noAsk bool, autoAcceptDefaults bool, ignoreMissingValues bool, answerQuestions func(
		console *tests.
			ConsoleWrapper, donec chan struct{})) (string, []*GeneratedSecret, error) {
	tests.SkipForWindows(t, "go-expect does not work on windows")
	t.Parallel()
	secrets := make([]*GeneratedSecret, 0)
	console := tests.NewTerminal(t)
	options := surveyutils.JSONSchemaOptions{
		Out:                 console.Out,
		In:                  console.In,
		OutErr:              console.Err,
		AskExisting:         askExisting,
		AutoAcceptDefaults:  autoAcceptDefaults,
		NoAsk:               noAsk,
		IgnoreMissingValues: ignoreMissingValues,

		CreateSecret: func(name string, key string, value string, passthrough bool) (interface{}, error) {
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

	// Test interactive IO
	donec := make(chan struct{})
	go answerQuestions(console, donec)
	assert.NoError(t, err)
	result, runErr := options.GenerateValues(
		data,
		existingValues)
	err = console.Close()
	<-donec
	assert.NoError(t, err)
	yaml, err := yaml.JSONToYAML(result)
	t.Logf(expect.StripTrailingEmptyLines(console.CurrentState()))
	assert.NoError(t, err)
	return string(yaml), secrets, runErr
}
