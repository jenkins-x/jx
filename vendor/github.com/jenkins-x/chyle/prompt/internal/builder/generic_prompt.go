package builder

// GenericPrompt defines a generic overridable prompt
type GenericPrompt struct {
	PromptID   string
	PromptStr  string
	OnSuccess  func(string) string
	OnError    func(error) string
	ParseValue func(string) error
}

// ID return ID of current prompt
func (t *GenericPrompt) ID() string {
	return t.PromptID
}

// PromptString returns string given by prompt
func (t *GenericPrompt) PromptString() string {
	return t.PromptStr
}

// Parse handles prompt value
func (t *GenericPrompt) Parse(value string) error {
	return t.ParseValue(value)
}

// NextOnSuccess returns the next prompt to reach when succeed
func (t *GenericPrompt) NextOnSuccess(value string) string {
	return t.OnSuccess(value)
}

// NextOnError returns an error when something wrong occurred
func (t *GenericPrompt) NextOnError(err error) string {
	return t.OnError(err)
}
