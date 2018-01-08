package survey

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/AlecAivazis/survey.v1/core"
)

func init() {
	// disable color output for all prompts to simplify testing
	core.DisableColor = true
}

func TestValidationError(t *testing.T) {

	err := fmt.Errorf("Football is not a valid month")

	actual, err := core.RunTemplate(
		core.ErrorTemplate,
		err,
	)
	if err != nil {
		t.Errorf("Failed to run template to format error: %s", err)
	}

	expected := `✘ Sorry, your reply was invalid: Football is not a valid month
`

	if actual != expected {
		t.Errorf("Formatted error was not formatted correctly. Found:\n%s\nExpected:\n%s", actual, expected)
	}
}

func TestAsk_returnsErrorIfTargetIsNil(t *testing.T) {
	// pass an empty place to leave the answers
	err := Ask([]*Question{}, nil)

	// if we didn't get an error
	if err == nil {
		// the test failed
		t.Error("Did not encounter error when asking with no where to record.")
	}
}

func TestPagination_tooFew(t *testing.T) {
	// a small list of options
	choices := []string{"choice1", "choice2", "choice3"}

	// a page bigger than the total number
	pageSize := 4
	// the current selection
	sel := 3

	// compute the page info
	page, idx := paginate(pageSize, choices, sel)

	// make sure we see the full list of options
	assert.Equal(t, choices, page)
	// with the second index highlighted (no change)
	assert.Equal(t, 3, idx)
}

func TestPagination_firstHalf(t *testing.T) {
	// the choices for the test
	choices := []string{"choice1", "choice2", "choice3", "choice4", "choice5", "choice6"}

	// section the choices into groups of 4 so the choice is somewhere in the middle
	// to verify there is no displacement of the page
	pageSize := 4
	// test the second item
	sel := 2

	// compute the page info
	page, idx := paginate(pageSize, choices, sel)

	// we should see the first three options
	assert.Equal(t, choices[0:4], page)
	// with the second index highlighted
	assert.Equal(t, 2, idx)
}

func TestPagination_middle(t *testing.T) {
	// the choices for the test
	choices := []string{"choice0", "choice1", "choice2", "choice3", "choice4", "choice5"}

	// section the choices into groups of 3
	pageSize := 2
	// test the second item so that we can verify we are in the middle of the list
	sel := 3

	// compute the page info
	page, idx := paginate(pageSize, choices, sel)

	// we should see the first three options
	assert.Equal(t, choices[2:4], page)
	// with the second index highlighted
	assert.Equal(t, 1, idx)
}

func TestPagination_lastHalf(t *testing.T) {
	// the choices for the test
	choices := []string{"choice0", "choice1", "choice2", "choice3", "choice4", "choice5"}

	// section the choices into groups of 3
	pageSize := 3
	// test the last item to verify we're not in the middle
	sel := 5

	// compute the page info
	page, idx := paginate(pageSize, choices, sel)

	// we should see the first three options
	assert.Equal(t, choices[3:6], page)
	// we should be at the bottom of the list
	assert.Equal(t, 2, idx)
}
