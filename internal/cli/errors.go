package cli

import "fmt"

// CommandError provides structured error reporting for CLI commands.
type CommandError struct {
	Message    string
	Cause      error
	Suggestion string
	ExitCode   int
}

// Error implements the error interface.
func (e CommandError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	if e.Cause != nil {
		return e.Cause.Error()
	}
	return "command failed"
}

// Unwrap exposes the wrapped error.
func (e CommandError) Unwrap() error {
	return e.Cause
}

// ExitStatus returns the process exit code associated with the error.
func (e CommandError) ExitStatus() int {
	if e.ExitCode != 0 {
		return e.ExitCode
	}
	return 1
}

// newCommandError builds a CommandError with formatted messaging.
func newCommandError(message string, cause error, suggestion string, exitCode int) CommandError {
	msg := message
	if cause != nil && message == "" {
		msg = cause.Error()
	}
	return CommandError{
		Message:    msg,
		Cause:      cause,
		Suggestion: suggestion,
		ExitCode:   exitCode,
	}
}

// wrapError is a helper to create a CommandError as an error interface.
func wrapError(message string, cause error, suggestion string, exitCode int) error {
	if cause == nil {
		return CommandError{Message: message, Suggestion: suggestion, ExitCode: exitCode}
	}
	return newCommandError(message, cause, suggestion, exitCode)
}

// formatSuggestion formats a hint for display when suggestions are provided.
func formatSuggestion(hint string) string {
	if hint == "" {
		return ""
	}
	return fmt.Sprintf("hint: %s", hint)
}
