package opts

// GetOptions is the start of the data required to perform the operation.  As new fields are added, add them here instead of
// referencing the cmd.Flags()
type StepOptions struct {
	*CommonOptions

	DisableImport bool
	OutDir        string
}

// Run implements this command
func (o *StepOptions) Run() error {
	return o.Cmd.Help()
}
