//-----------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation.  All rights reserved.
//-----------------------------------------------------------------------------

namespace CnsWrapperService
{
    class TelemetrySettingsModel
    {
        // Flag to disable the telemetry.
        public bool DisableAll { get; set; }

        // Flag to Disable sending trace.
        public bool DisableTrace { get; set; }

        // Flag to Disable sending metric.
        public bool DisableMetric { get; set; }

        // Flag to Disable sending events.
        public bool DisableEvent { get; set; }

        // Configure how many bytes can be sent in one call to the data collector
        public int TelemetryBatchSizeBytes { get; set; }

        // Configure the maximum delay before sending queued telemetry in milliseconds
        public int TelemetryBatchIntervalInSecs { get; set; }

        // Heartbeat interval for sending heartbeat metric
        public int HeartBeatIntervalInMins { get; set; }

        // Enable thread for getting metadata from wireserver
        public bool DisableMetadataRefreshThread { get; set; }

        // Refresh interval in milliseconds for metadata thread
        public int RefreshIntervalInSecs { get; set; }

        // Disable debug logging for telemetry messages
        public bool DebugMode { get; set; }

        // Interval for sending snapshot events.
        public int SnapshotIntervalInMins { get; set; }
    }
}
