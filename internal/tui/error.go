package tui

type LoadError struct {
	Err         error
	Diagnostics LoadDiagnostics
}

func (e LoadError) Error() string {
	return e.Err.Error()
}

func (e LoadError) Unwrap() error {
	return e.Err
}
