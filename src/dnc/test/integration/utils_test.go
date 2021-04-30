package k8s

import (
	"context"
	crd "dnc/requestcontroller/kubernetes"
	"dnc/test/k8s"
	"testing"

	"github.com/stretchr/testify/require"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func mustLabelSwiftNodes(t *testing.T, ctx context.Context, clientset *kubernetes.Clientset, delegatedSubnetID, delegatedSubnetName string) {
	swiftNodeLabels := map[string]string{
		crd.DelegatedSubnetIDLabel: delegatedSubnetID,
		crd.SubnetNameLabel:        delegatedSubnetName,
	}

	res, err := clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		t.Fatalf("could not list nodes: %v", err)
	}
	for _, node := range res.Items {
		_, err := k8s.AddNodeLabels(ctx, clientset.CoreV1().Nodes(), node.Name, swiftNodeLabels)
		if err != nil {
			t.Fatalf("could not add labels to node: %v", err)
		}
		t.Logf("labels added to node %s", node.Name)
	}
}

func mustSetUpRBAC(t *testing.T, ctx context.Context, k8sShim *k8s.Shim) (cleanUpFunc func(t *testing.T)) {
	clusterRole, err := k8s.ParseClusterRole("testdata/goldpinger/cluster-role.yaml")
	require.NoError(t, err)
	clusterRoleBinding, err := k8s.ParseClusterRoleBinding("testdata/goldpinger/cluster-role-binding.yaml")
	require.NoError(t, err)
	serviceAccount, err := k8s.ParseServiceAccount("testdata/goldpinger/service-account.yaml")
	require.NoError(t, err)

	err = k8sShim.CreateOrReplaceServiceAccount(ctx, serviceAccount.Namespace, serviceAccount)
	require.NoError(t, err)
	err = k8sShim.CreateOrReplaceClusterRole(ctx, clusterRole)
	require.NoError(t, err)
	err = k8sShim.CreateOrReplaceClusterRoleBinding(ctx, clusterRoleBinding)
	require.NoError(t, err)

	t.Log("rbac set up")

	return func(t *testing.T) {
		t.Log("cleaning up rbac")

		if err := k8sShim.DeleteServiceAccount(ctx, serviceAccount.Namespace, serviceAccount.Name); err != nil {
			t.Log(err)
		}
		if err := k8sShim.DeleteClusterRoleBinding(ctx, clusterRoleBinding.Name); err != nil {
			t.Log(err)
		}
		if err := k8sShim.DeleteClusterRole(ctx, clusterRole.Name); err != nil {
			t.Log(err)
		}

		t.Log("rbac cleaned up")
	}
}
