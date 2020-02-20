// Copyright Microsoft. All rights reserved.

package telemetry

const (

	// Metric Names
	CNIExecutimeMetricStr   = "CNIExecutionTimeMs"
	CNIV6ExecutimeMetricStr = "CNIV6ExecutionTimeMs"
	CNIAddTimeMetricStr     = "CNIAddTimeMs"
	CNIDelTimeMetricStr     = "CNIDelTimeMs"
	CNIUpdateTimeMetricStr  = "CNIUpdateTimeMs"

	// Dimension Names
	ContextStr        = "Context"
	SubContextStr     = "SubContext"
	VMUptimeStr       = "VMUptime"
	OperationTypeStr  = "OperationType"
	VersionStr        = "Version"
	StatusStr         = "Status"
	CNIModeStr        = "CNIMode"
	CNINetworkModeStr = "CNINetworkMode"
	OSTypeStr         = "OSType"

	// Values
	SucceededStr     = "Succeeded"
	FailedStr        = "Failed"
	SingleTenancyStr = "SingleTenancy"
	MultiTenancyStr  = "MultiTenancy"
)
