package pubsubproxy

import (
	"dnc/api"
	"dnc/datamodel"
)

func GetPendingNCJobContexts(NetworkID string, NCID string) ([]datamodel.NcPendingPublishRequestContext, *api.JsonErr) {
	pendingPublishContexts := []datamodel.NcPendingPublishRequestContext{}
	for pubsubAzId, _ := range getAllPublishers() {
		nckey := datamodel.GetNcPendingPublishContextKey(NetworkID, NCID, pubsubAzId)
		ncPendingPublishContext, err := datamodel.NcPendingPublishRepo.TryGetNcPendingPublish(nckey)
		if err != nil {
			return nil, err
		}
		if ncPendingPublishContext != nil {
			pendingPublishContexts = append(pendingPublishContexts, *ncPendingPublishContext)
		} else {
			// todo: inject logger. this makes tests for this function difficult to wire.
			// loggershim.LS.Errorf("Error: pending NC Job with NC Key %s not found", nckey)
		}
	}

	return pendingPublishContexts, nil
}
