package k8s

import (
	"context"
	"dnc/retry"
	"dnc/test/integration/goldpinger"
	"dnc/test/k8s"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/homedir"
)

var (
	defaultRetrier = retry.Retrier{Attempts: 15, Delay: time.Second}
	// todo: could use another method for more robust kubeconfig loading.
	// See https://github.com/kubernetes-sigs/controller-runtime/blob/e10bf72058d349e9c5cdd177593c32c650065458/pkg/client/config/config.go#L59
	kubeconfig          = flag.String("test-kubeconfig", filepath.Join(homedir.HomeDir(), ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	delegatedSubnetID   = flag.String("delegated-subnet-id", "", "delegated subnet id for node labeling")
	delegatedSubnetName = flag.String("subnet-name", "", "subnet name for node labeling")
)

const (
	subnetIDNodeLabelEnvVar   = "DELEGATED_SUBNET_ID_NODE_LABEL"
	subnetNameNodeLabelEnvVar = "SUBNET_NAME_NODE_LABEL"
)

func shouldLabelNodes() bool {
	if *delegatedSubnetID == "" {
		*delegatedSubnetID = os.Getenv(subnetIDNodeLabelEnvVar)
	}
	if *delegatedSubnetName == "" {
		*delegatedSubnetName = os.Getenv(subnetNameNodeLabelEnvVar)
	}
	return *delegatedSubnetID != "" && *delegatedSubnetName != ""
}

/*

In order to run the tests below, you need a k8s cluster and its kubeconfig.
If no kubeconfig is passed, the test will attempt to find one in the default location for kubectl config.
The test will also attempt to label the nodes if the appropriate flags or environment variables are set.
Run the tests as follows:

go test -v . [-args test-kubeconfig=<...> delegated-subnet-id=<...> subnet-name=<...>]

todo: consider adding the following scenarios
- [x] All pods should be assigned an IP.
- [ ] All pod IPs should belong to the delegated subnet and not overlap host subnet.
- [x] All pods should be able to ping each other.
- [ ] All pods should be able to ping nodes. Daemonset with hostnetworking?
- [ ] All pods should be able to reach public internet. Enable hosts to ping in goldpinger deployment.

- [x] All scenarios above should be valid during deployment scale up
- [x] All scenarios above should be valid during deployment scale down

- [ ] All scenarios above should be valid during node scale up (i.e more nodes == more NNCs)
- [ ] All scenarios above should be valid during node scale down

todo:
  - Need hook for `az aks scale --g <resource group> --n <cluster name> --node-count <prev++> --nodepool-name <np name>`
  - Need hook for pubsub client to verify that no secondary CAs are leaked
  - Check CNS ipam pool?
  - Check NNC in apiserver?
*/

func TestPodScaling(t *testing.T) {
	clientset, err := k8s.GetClientset(*kubeconfig)
	require.NoError(t, err)
	restConfig, err := k8s.GetRestConfig(*kubeconfig)
	require.NoError(t, err)
	deployment, err := k8s.ParseDeployment("testdata/goldpinger/deployment.yaml")
	require.NoError(t, err)

	ctx := context.Background()

	if shouldLabelNodes() {
		mustLabelSwiftNodes(t, ctx, clientset, *delegatedSubnetID, *delegatedSubnetName)
	} else {
		t.Log("swift node labels not passed or set. skipping labeling")
	}

	k8sShim := k8s.NewShim(clientset)

	rbacCleanUpFn := mustSetUpRBAC(t, ctx, k8sShim)
	err = k8sShim.CreateOrReplaceDeployment(ctx, deployment.Namespace, deployment)
	require.NoError(t, err)

	t.Cleanup(func() {
		t.Log("cleaning up resources")
		rbacCleanUpFn(t)

		if err := k8sShim.DeleteDeployment(ctx, deployment.Namespace, deployment.Name); err != nil {
			t.Log(err)
		}
	})

	counts := []int{10, 20, 50, 10}

	for _, c := range counts {
		count := c
		t.Run(fmt.Sprintf("replica count %d", count), func(t *testing.T) {
			replicaCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
			defer cancel()

			scaleFunc := func() error {
				return k8sShim.ScaleDeployment(replicaCtx, deployment.Namespace, deployment.Name, count)
			}
			if err := defaultRetrier.Do(replicaCtx, scaleFunc); err != nil {
				t.Fatalf("could not scale deployment: %v", err)
			}

			t.Run("all pods have IPs assigned", func(t *testing.T) {
				podsClient := clientset.CoreV1().Pods(deployment.Namespace)

				checkPodIPsFn := func() error {
					podList, err := podsClient.List(ctx, metav1.ListOptions{LabelSelector: "app=goldpinger"})
					if err != nil {
						return err
					}

					if len(podList.Items) == 0 {
						return errors.New("no pods scheduled")
					}

					for _, pod := range podList.Items {
						if pod.Status.Phase == apiv1.PodPending {
							return errors.New("some pods still pending")
						}
					}

					for _, pod := range podList.Items {
						if pod.Status.PodIP == "" {
							return errors.New("a pod has not been allocated an IP")
						}
					}

					return nil
				}
				err := defaultRetrier.Do(ctx, checkPodIPsFn)
				if err != nil {
					t.Fatalf("not all pods were allocated IPs: %v", err)
				}
				t.Log("all pods have been allocated IPs")
			})

			t.Run("all pods can ping each other", func(t *testing.T) {
				pf, err := k8s.NewPortForwarder(restConfig)
				if err != nil {
					t.Fatal(err)
				}

				portForwardCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
				defer cancel()

				var streamHandle k8s.PortForwardStreamHandle
				portForwardFn := func() error {
					t.Log("attempting port forward")
					handle, err := pf.Forward(ctx, "default", "app=goldpinger", 8080, 8080)
					if err != nil {
						return err
					}
					streamHandle = handle
					return nil
				}
				if err := defaultRetrier.Do(portForwardCtx, portForwardFn); err != nil {
					t.Fatalf("could not start port forward within 30s: %v", err)
				}
				defer streamHandle.Stop()

				gpClient := goldpinger.Client{Host: streamHandle.Url()}

				clusterCheckCtx, cancel := context.WithTimeout(ctx, time.Minute)
				defer cancel()
				clusterCheckFn := func() error {
					clusterState, err := gpClient.CheckAll(clusterCheckCtx)
					if err != nil {
						return err
					}

					stats := goldpinger.ClusterStats(clusterState)
					stats.PrintStats()
					if stats.AllPingsHealthy() {
						return nil
					}

					return errors.New("not all pings are healthy")
				}
				retrier := retry.Retrier{Attempts: 10, Delay: 5 * time.Second}
				if err := retrier.Do(clusterCheckCtx, clusterCheckFn); err != nil {
					t.Fatalf("cluster could not reach healthy state: %v", err)
				}

				t.Log("all pings successful!")
			})
		})
	}
}
