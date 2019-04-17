package logger

import "regexp"

const (
	ErrorCodePrefix = 5
)

var CompiledRegex = regexp.MustCompile(`Code:(\w*)`)
