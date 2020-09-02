package main

import acncommon "github.com/Azure/azure-container-networking/common"

const (
	OptVersion              = "version"
	OptVersionAlias         = "v"
	OptLogTargetStderr      = "standarderror"
	OptLogTargetStderrAlias = "stderr"
)

var args = acncommon.ArgumentList{
	{
		Name:         OptVersion,
		Shorthand:    OptVersionAlias,
		Description:  "Print version information",
		Type:         "bool",
		DefaultValue: false,
	},
	{
		Name:         OptLogTargetStderr,
		Shorthand:    OptLogTargetStderrAlias,
		Description:  "Set the log target to stdout.",
		Type:         "bool",
		DefaultValue: false,
	},
}
