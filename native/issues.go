package main

import "slices"

// Keep diagnostic details in the issues tab instead of the status bar.
func (a *application) reportError(message string) {
	if !slices.Contains(a.operationErrors, message) {
		a.operationErrors = append(a.operationErrors, message)
	}
	a.status = "操作失败"
	a.statusError = true
	a.resultTab = 2
}

func (a *application) issueErrors() []string {
	var messages []string
	if a.result != nil {
		messages = append(messages, a.result.Errors...)
	}
	messages = append(messages, a.operationErrors...)
	// An untouched empty form is not an error. Once entries are present,
	// disabled extraction and its explanation share the same diagnostic tab.
	if len(lines(a.config.Entries)) > 0 {
		if _, err := a.config.request(); err != nil && !slices.Contains(messages, err.Error()) {
			messages = append(messages, err.Error())
		}
	}
	return messages
}

func (a *application) issueCount() int {
	n := len(a.issueErrors())
	if a.result != nil {
		n += len(a.result.Infos)
	}
	return n
}
