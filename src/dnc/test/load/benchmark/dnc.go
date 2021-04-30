package benchmark

import (
	"context"
	"dnc/api"
	"dnc/async"
	"dnc/client"
	"dnc/datamodel"
	"dnc/retry"
	"errors"
	"math"
	"net/http"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

var defaultRetrier = retry.Retrier{
	Attempts:   10,
	Delay:      time.Second,
	ExpBackoff: false,
}

type DNC struct {
	Client client.DncClient
}

/*
  todo: methods should return errors instead of fataling.
   will handle with errgroup which makes it easier but needs to be vendored
*/

func (d *DNC) CreateVnets(vnetIDs []string) []time.Duration {
	var ds []time.Duration
	for _, vnetID := range vnetIDs {
		start := time.Now()
		if err := d.Client.AddNetwork(vnetID); err.HTTPStatus != http.StatusOK {
			logrus.Fatalf("could not create vnet: %v", err)
		}
		logrus.Infof("vnet %s posted", vnetID)
		ds = append(ds, time.Since(start))
	}
	return ds
}

func (d *DNC) JoinSubnets(ctx context.Context, vnetIDToSubnets map[string][]string) []time.Duration {
	var ds []time.Duration

	for vnetID, subnets := range vnetIDToSubnets {
		for _, subnet := range subnets {

			subnetJoinFunc := func() error {
				if err := d.Client.AddSubnet(vnetID, subnet, "foo"); err.HTTPStatus != http.StatusOK {
					logrus.Errorf("could not persist vnet %s subnet %s, retrying: %v", vnetID, subnet, err)
					return err
				}
				return nil
			}

			start := time.Now()
			// requires retries if using prepopulated allocs, for example
			if err := defaultRetrier.Do(ctx, subnetJoinFunc); err != nil {
				logrus.Fatalf("could not persist vnet %s subnet %s: %v", vnetID, subnet, err)
			}
			end := time.Since(start)

			logrus.Infof("joined vnet %s subnet %s, took %v", vnetID, subnet, end)
			ds = append(ds, end)
		}
	}

	return ds
}

func (d *DNC) CreateNodes(ctx context.Context, workerCount int, nodeRequests []NodeRequest) []time.Duration {
	var (
		nodeCreationTimes durations
		wg                sync.WaitGroup
	)

	nodeReqCh := streamNodeRequests(nodeRequests)
	wg.Add(workerCount)
	for i := 0; i < workerCount; i++ {
		go func() {
			defer wg.Done()
			for req := range nodeReqCh {
				createNodeFunc := func() error {
					err := d.Client.AddNode(req.NodeID, req.NodeReq.IPAddresses[0], req.NodeReq.OrchestratorType, req.NodeReq.InfrastructureNetwork, "", "")
					if err.HTTPStatus != http.StatusOK {
						return err
					}
					return nil
				}

				nodeCreateStart := time.Now()
				if err := defaultRetrier.Do(ctx, createNodeFunc); err != nil {
					logrus.Fatalf("could not create node %s: %v", req.NodeID, err)
				}

				nodeCreationTimes.Add(time.Since(nodeCreateStart))
			}
		}()
	}

	wg.Wait()

	return nodeCreationTimes.GetAll()
}

func (d *DNC) CreateNCs(ctx context.Context, workerCount int, ncRequests []NCRequest) []time.Duration {
	var (
		ncCreationTimes durations
		wg              sync.WaitGroup
	)

	cancelLog := async.IntervalDo(time.Second*10, func() {
		logrus.Infof("created %d ncs", ncCreationTimes.Len())
	})
	defer cancelLog()

	ncCreateRetry := retry.Retrier{Attempts: 10, Delay: time.Second}
	ncReqCh := streamNcRequests(ncRequests)
	wg.Add(workerCount)
	for i := 0; i < workerCount; i++ {
		go func() {
			defer wg.Done()
			for req := range ncReqCh {
				ncCreateStart := time.Now()
				err := ncCreateRetry.Do(ctx, func() error {
					err := d.Client.UpsertNcV2(req.VnetID, req.NCID, req.NCReq)
					if err.HTTPStatus != http.StatusOK {
						return err
					}
					return nil
				})
				if err != nil {
					logrus.Fatalf("could not create nc %s: %v", req.NCID, err)
				}

				ncCreationTimes.Add(time.Since(ncCreateStart))
			}
		}()
	}

	wg.Wait()

	return ncCreationTimes.GetAll()
}

func (d *DNC) GetAllNCResponses(ctx context.Context, workerCount int, ncRequests []NCRequest) []NCResponse {
	var (
		responses ncResponses
		wg        sync.WaitGroup
	)

	collectCtx, cancel := context.WithTimeout(ctx, time.Minute*10)
	defer cancel()

	cancelLog := async.IntervalDo(time.Second*10, func() {
		logrus.Infof("gathered %d nc responses", responses.Len())
	})
	defer cancelLog()

	ncVerifyCh := streamNcRequests(ncRequests)
	wg.Add(workerCount)
	for i := 0; i < workerCount; i++ {
		go func() {
			defer wg.Done()
			for req := range ncVerifyCh {
				stats, err := d.getNCResponse(collectCtx, req.VnetID, req.NCID)
				if err != nil {
					logrus.Errorf("could not verify nc %s: %v", req.NCID, err)
					continue
				}
				responses.Add(stats)
			}
		}()
	}

	wg.Wait()
	return responses.GetAll()
}

var ncRetrier = retry.Retrier{
	Attempts:   math.MaxInt64, // attempt until context cancellation
	Delay:      time.Second * 3,
	ExpBackoff: false,
}

func (d *DNC) getNCResponse(ctx context.Context, vnetID, ncID string) (NCResponse, error) {
	var (
		nc     *api.NetworkContainerResponse
		apiErr *api.JsonErr
	)

	retryFunc := func() error {
		nc, apiErr = d.Client.GetNC(vnetID, ncID)
		if apiErr.HTTPStatus != http.StatusOK {
			if apiErr.HTTPStatus == http.StatusNotFound {
				return nil // don't retry
			}
			return apiErr
		}

		if nc.RequestStatus.Status == datamodel.Pending.String() {
			return errors.New("still pending, retry")
		}

		return nil
	}

	if err := ncRetrier.Do(ctx, retryFunc); err != nil { // we expect error only from timeout
		if apiErr != nil { // last captured api error, if available
			return NCResponse{}, apiErr
		}
		return NCResponse{}, err // never received a response
	}

	if apiErr != nil && apiErr.HTTPStatus == http.StatusNotFound {
		return NCResponse{}, apiErr
	}

	return ncResponseFromNC(*nc), nil
}
