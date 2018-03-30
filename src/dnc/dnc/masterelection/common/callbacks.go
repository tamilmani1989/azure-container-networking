package callbacks

// ReplicaChangeNotification - Delegate for replica change notification.
type ReplicaChangeNotification func(bool)

// PrimaryIPEndpointNotification - Delegate for primary IP Endpoint notification.
type PrimaryIPEndpointNotification func(string)
