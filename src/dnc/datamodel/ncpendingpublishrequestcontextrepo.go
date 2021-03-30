package datamodel

import (
	"dnc/api"
)
//TODO: Change from wrapper to having the actual implementation https://msazure.visualstudio.com/One/_workitems/edit/9570989
type NcPendingPublishRequestContextRepo struct{}

func (*NcPendingPublishRequestContextRepo) GetNcPendingPublishRequestContexts() (map[string]NcPendingPublishRequestContext, error) {
	return GetNcPendingPublishRequestContexts()
}

func (*NcPendingPublishRequestContextRepo) RemoveNcPendingPublishContext(networkID, networkContainerID, pubsubAzID string) *api.JsonErr {
	return RemoveNcPendingPublishContext(networkID, networkContainerID, pubsubAzID)
}

func (*NcPendingPublishRequestContextRepo) CreateOrUpdateNcPendingPublishRequest(requestContext *NcPendingPublishRequestContext) *api.JsonErr {
	return CreateOrUpdateNcPendingPublishRequest(requestContext)
}

func (*NcPendingPublishRequestContextRepo) QueryNcPendingPublishContext(key string) (*NcPendingPublishRequestContext, *api.JsonErr) {
	return QueryNcPendingPublishContext(key)
}
