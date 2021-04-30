package k8s

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
)

// ParseDeployment parses a appsv1.Deployment from the provided yaml or json file path.
func ParseDeployment(path string) (appsv1.Deployment, error) {
	var depl appsv1.Deployment
	if err := ParseResource(path, &depl); err != nil {
		return appsv1.Deployment{}, err
	}
	return depl, nil
}

// ParseClusterRole parses a rbacv1.ClusterRole from the provided yaml or json file path.
func ParseClusterRole(path string) (rbacv1.ClusterRole, error) {
	var cr rbacv1.ClusterRole
	if err := ParseResource(path, &cr); err != nil {
		return rbacv1.ClusterRole{}, err
	}
	return cr, nil
}

// ParseClusterRoleBinding parses a rbacv1.ClusterRoleBinding from the provided yaml or json file path.
func ParseClusterRoleBinding(path string) (rbacv1.ClusterRoleBinding, error) {
	var crb rbacv1.ClusterRoleBinding
	if err := ParseResource(path, &crb); err != nil {
		return rbacv1.ClusterRoleBinding{}, err
	}
	return crb, nil
}

// ParseServiceAccount parses a corev1.ServiceAccount from the provided yaml or json file path.
func ParseServiceAccount(path string) (corev1.ServiceAccount, error) {
	var svcAcct corev1.ServiceAccount
	if err := ParseResource(path, &svcAcct); err != nil {
		return corev1.ServiceAccount{}, err
	}
	return svcAcct, nil
}
