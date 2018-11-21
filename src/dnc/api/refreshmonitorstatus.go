package api

import (
	"fmt"
)

// RefreshMonitorStatus - Struct to hold refresh monitor data
type RefreshMonitorStatus struct {
	StartTime           string `json:"StartTime"`
	EndTime             string `json:"EndTime"`
	IsRefreshInProgress bool   `json:"IsRefreshInProgress"`
	IsThrottled         bool   `json:"IsThrottled"`
	ProcessedItems      int    `json:"ProcessedItems"`
	FailedItems         int    `json:"FailedItems"`
	TotalItems          int    `json:"TotalItems"`
	TimeToRefresh       string `json:"TimeToRefresh"`
}

func (refreshMonitorStatus *RefreshMonitorStatus) String() string {
	return fmt.Sprintf(
		"RefreshMonitorStatus:[StartTime:%s, EndTime:%s,IsRefreshInProgress:%v, TimeToRefresh:%v, IsThrottled:%v, ProcessedItems:%v, TotalItems:%v, FailedItems:%v]",
		refreshMonitorStatus.StartTime, refreshMonitorStatus.EndTime, refreshMonitorStatus.IsRefreshInProgress, refreshMonitorStatus.TimeToRefresh,
		refreshMonitorStatus.IsThrottled, refreshMonitorStatus.ProcessedItems, refreshMonitorStatus.TotalItems, refreshMonitorStatus.FailedItems)
}
