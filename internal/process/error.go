package process

import "os/exec"

// Error builds on exec.ExitError and adds the original command as well the full combined output.
type Error struct {
	*exec.ExitError

	combined []byte
	command  string
}

func NewError(err *exec.ExitError, combined []byte, command string) *Error {
	return &Error{
		ExitError: err,
		combined:  combined,
		command:   command,
	}
}

func (pe *Error) Combined() []byte {
	return pe.combined
}

func (pe *Error) Command() string {
	return pe.command
}

func (pe *Error) Unwrap() error {
	return pe.ExitError
}
