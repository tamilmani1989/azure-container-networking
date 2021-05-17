package kubernetes

import (
	"context"
	"dnc/retry"
	"fmt"
	"reflect"
	"strings"
	"time"

	acnv1alpha "github.com/Azure/azure-container-networking/nodenetworkconfig/api/v1alpha"
	"github.com/sirupsen/logrus"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apiextensionsclientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
)

// K8sCRDClient is an interface that wraps only the functions of a k8s crd client that the controller will use to ensure the CRD is installed
// on the cluster. Unit tests can implement this interface to mock a K8s CRD Client
type K8sCRDClient interface {
	Create(context.Context, *apiextensionsv1.CustomResourceDefinition, metav1.CreateOptions) (*apiextensionsv1.CustomResourceDefinition, error)
}

// getCRDClientforKubeconfig gets a K8sCRDClient for a given kubeconfig. Have to return a K8sClient (interface) here instead of a struct since the k8s libs
// hide the structs from the caller
func getCRDClientForRESTConfig(restConfig *rest.Config) (K8sCRDClient, error) {
	crdClientSet, err := apiextensionsclientset.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("Error creating client for CRD definition creation: %w", err)
	}

	return crdClientSet.ApiextensionsV1().CustomResourceDefinitions(), nil
}

func ensureCRDIsRegistered(ctx context.Context, client K8sCRDClient, logger logrus.FieldLogger) error {
	logger.Info("ensuring CRD definition exists on cluster")
	retrier := retry.Retrier{
		Attempts: 60,
		Delay:    time.Second * 5,
	}

	return retrier.Do(ctx, func() error {
		crd := &apiextensionsv1.CustomResourceDefinition{
			ObjectMeta: metav1.ObjectMeta{
				Name: nncPluralName + "." + acnv1alpha.GroupVersion.Group,
			},
			Spec: apiextensionsv1.CustomResourceDefinitionSpec{
				Group: acnv1alpha.GroupVersion.Group,
				Versions: []apiextensionsv1.CustomResourceDefinitionVersion{
					{
						Name:    acnv1alpha.GroupVersion.Version,
						Served:  true,
						Storage: true,
						Subresources: &apiextensionsv1.CustomResourceSubresources{
							Status: &apiextensionsv1.CustomResourceSubresourceStatus{},
						},
						Schema: &apiextensionsv1.CustomResourceValidation{
							OpenAPIV3Schema: &apiextensionsv1.JSONSchemaProps{
								Description: "NodeNetworkConfig is the Schema for the nodenetworkconfigs API",
								Type:        "object",
								Properties: map[string]apiextensionsv1.JSONSchemaProps{
									"apiVersion": {
										Description: "APIVersion defines the versioned schema of this representation",
										Type:        "string",
									},
									"kind": {
										Description: "Kind is a string value representing the REST resource this object represents",
										Type:        "string",
									},
									"metadata": {
										Type: "object",
									},
									"spec": {
										Type:        "object",
										Description: "NodeNetworkConfigSpec defines the desired state of NetworkConfig",
										Properties: map[string]apiextensionsv1.JSONSchemaProps{
											"ipsNotInUse": {
												Items: &apiextensionsv1.JSONSchemaPropsOrArray{
													Schema: &apiextensionsv1.JSONSchemaProps{
														Type: "string",
													},
												},
												Type: "array",
											},
											"requestedIPCount": {
												Format: "int64",
												Type:   "integer",
											},
										},
									},
									"status": {
										Type:        "object",
										Description: "NodeNetworkConfigStatus defines the observed state of NetworkConfig",
										Properties: map[string]apiextensionsv1.JSONSchemaProps{
											"networkContainers": {
												Type: "array",
												Items: &apiextensionsv1.JSONSchemaPropsOrArray{
													Schema: &apiextensionsv1.JSONSchemaProps{
														Type: "object",
														Properties: map[string]apiextensionsv1.JSONSchemaProps{
															"defaultGateway": {
																Type: "string",
															},
															"ipAssignments": {
																Type: "array",
																Items: &apiextensionsv1.JSONSchemaPropsOrArray{
																	Schema: &apiextensionsv1.JSONSchemaProps{
																		Type: "object",
																		Properties: map[string]apiextensionsv1.JSONSchemaProps{
																			"ip": {
																				Type: "string",
																			},
																			"name": {
																				Type: "string",
																			},
																		},
																	},
																},
															},
															"id": {
																Type: "string",
															},
															"primaryIP": {
																Type: "string",
															},
															"subnetAddressSpace": {
																Type: "string",
															},
															"subnetName": {
																Type: "string",
															},
															"version": {
																Type:   "integer",
																Format: "int64",
															},
														},
													},
												},
											},
											"scaler": {
												Description: "Scaler groups IP request params together",
												Type:        "object",
												Properties: map[string]apiextensionsv1.JSONSchemaProps{
													"batchSize": {
														Type:   "integer",
														Format: "int64",
													},
													"releaseThresholdPercent": {
														Type:   "integer",
														Format: "int64",
													},
													"requestThresholdPercent": {
														Type:   "integer",
														Format: "int64",
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
				Scope: apiextensionsv1.NamespaceScoped,
				Names: apiextensionsv1.CustomResourceDefinitionNames{
					Plural:     nncPluralName,
					Kind:       reflect.TypeOf(acnv1alpha.NodeNetworkConfig{}).Name(),
					ListKind:   reflect.TypeOf(acnv1alpha.NodeNetworkConfig{}).Name() + "List",
					Singular:   strings.ToLower(reflect.TypeOf(acnv1alpha.NodeNetworkConfig{}).Name()),
					ShortNames: []string{nncShortName},
				},
			},
		}

		_, err := client.Create(ctx, crd, metav1.CreateOptions{})

		if err != nil && !apierrors.IsAlreadyExists(err) {
			logger.Errorf("could not add CRD definition to cluster: %v", err)
			return err
		}

		return nil
	})
}
