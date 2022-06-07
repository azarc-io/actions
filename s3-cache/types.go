package main

import "log"

const (
	// PutAction - Put artifacts
	PutAction = "put"

	// DeleteAction - Delete artifacts
	DeleteAction = "delete"

	// GetAction - Get artifacts
	GetAction = "get"

	// ErrCodeNotFound - s3 Not found error code
	ErrCodeNotFound = "NotFound"
)

type (
	// Action - Input params
	Action struct {
		Action    string
		Bucket    string
		S3Class   string
		Key       string
		Artifacts []string
	}
)

type MultiLevelLogger struct {
	verbose bool
}

func (l *MultiLevelLogger) Verbose(v ...interface{}) {
	if l.verbose {
		log.Print(v...)
	}
}
func (l *MultiLevelLogger) Warning(v ...interface{}) {
	log.Print(v...)
}
