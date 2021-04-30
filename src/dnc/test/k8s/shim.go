// Package k8s provides helper utilities for manipulation of k8s resources
package k8s

import (
	"context"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// Shim is a wrapper over a kubernetes.Clientset. It uses stable API versions and default options for most methods.
// For more fine grained control, kubernetes.Clientset should be used.
type Shim struct {
	Clientset *kubernetes.Clientset
}

// NewShim creates a new Shim
func NewShim(cs *kubernetes.Clientset) *Shim {
	return &Shim{Clientset: cs}
}

// todo: methods below are not exhaustive for different resources. add more as required.

// CreateOrReplaceDeployment creates the provided deployment or replaces any existing one with the same name.
func (s *Shim) CreateOrReplaceDeployment(ctx context.Context, namespace string, deployment appsv1.Deployment) error {
	if err := s.DeleteDeployment(ctx, namespace, deployment.Name); err != nil {
		if !apierrors.IsNotFound(err) {
			return err
		}
	}

	_, err := s.Clientset.AppsV1().Deployments(namespace).Create(ctx, &deployment, metav1.CreateOptions{})
	return err
}

// DeleteDeployment deletes a deployment.
func (s *Shim) DeleteDeployment(ctx context.Context, namespace, name string) error {
	if err := s.Clientset.AppsV1().Deployments(namespace).Delete(ctx, name, metav1.DeleteOptions{}); err != nil {
		if !apierrors.IsNotFound(err) {
			return err
		}
	}
	return nil
}

// ScaleDeployment scales a deployment.
func (s *Shim) ScaleDeployment(ctx context.Context, namespace, name string, replicas int) error {
	res, err := s.Clientset.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return err
	}

	res.Spec.Replicas = int32ptr(int32(replicas))
	_, err = s.Clientset.AppsV1().Deployments(namespace).Update(ctx, res, metav1.UpdateOptions{})
	return err
}

// CreateOrReplaceClusterRole creates the provided cluster role or replaces any existing one with the same name.
func (s *Shim) CreateOrReplaceClusterRole(ctx context.Context, clusterRole rbacv1.ClusterRole) error {
	if err := s.DeleteClusterRole(ctx, clusterRole.Name); err != nil {
		return err
	}

	_, err := s.Clientset.RbacV1().ClusterRoles().Create(ctx, &clusterRole, metav1.CreateOptions{})
	return err
}

// DeleteClusterRole deletes a cluster role.
func (s *Shim) DeleteClusterRole(ctx context.Context, name string) error {
	if err := s.Clientset.RbacV1().ClusterRoles().Delete(ctx, name, metav1.DeleteOptions{}); err != nil {
		if !apierrors.IsNotFound(err) {
			return err
		}
	}
	return nil
}

// CreateOrReplaceClusterRoleBinding creates the provided cluster role binding or replaces any existing one with the same name.
func (s *Shim) CreateOrReplaceClusterRoleBinding(ctx context.Context, clusterRoleBinding rbacv1.ClusterRoleBinding) error {
	if err := s.DeleteClusterRoleBinding(ctx, clusterRoleBinding.Name); err != nil {
		return err
	}
	_, err := s.Clientset.RbacV1().ClusterRoleBindings().Create(ctx, &clusterRoleBinding, metav1.CreateOptions{})
	return err
}

// DeleteClusterRoleBinding deletes a cluster role binding.
func (s *Shim) DeleteClusterRoleBinding(ctx context.Context, name string) error {
	if err := s.Clientset.RbacV1().ClusterRoleBindings().Delete(ctx, name, metav1.DeleteOptions{}); err != nil {
		if !apierrors.IsNotFound(err) {
			return err
		}
	}
	return nil
}

// CreateOrReplaceServiceAccount creates the provided service account or replaces any existing one with the same name.
func (s *Shim) CreateOrReplaceServiceAccount(ctx context.Context, namespace string, serviceAccount corev1.ServiceAccount) error {
	if err := s.DeleteServiceAccount(ctx, namespace, serviceAccount.Name); err != nil {
		if !apierrors.IsNotFound(err) {
			return err
		}
	}
	_, err := s.Clientset.CoreV1().ServiceAccounts(namespace).Create(ctx, &serviceAccount, metav1.CreateOptions{})
	return err
}

// DeleteServiceAccount deletes a service account.
func (s *Shim) DeleteServiceAccount(ctx context.Context, namespace, name string) error {
	if err := s.Clientset.CoreV1().ServiceAccounts(namespace).Delete(ctx, name, metav1.DeleteOptions{}); err != nil {
		if !apierrors.IsNotFound(err) {
			return err
		}
	}
	return nil
}

func int32ptr(i int32) *int32 { return &i }
