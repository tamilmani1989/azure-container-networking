// Copyright 2020 Microsoft. All rights reserved.

package kubernetes

import (
	"context"
	"encoding/json"
	"fmt"

	acnv1alpha "github.com/Azure/azure-container-networking/nodenetworkconfig/api/v1alpha"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

// mockK8sClient is a very minimal mock that stores objects in a map with a key equal to the objects "namespace/name". This is fine for request
// controller because it only interacts with nodes (no namespace) and NNCs (namespaced). This should not be used by any other project, and thus
// is package-private.
type mockK8sClient struct {
	objects map[string]runtime.Object
	scheme  *runtime.Scheme
}

func newMockK8sClient() *mockK8sClient {
	s := runtime.NewScheme()
	_ = clientgoscheme.AddToScheme(s)
	_ = acnv1alpha.AddToScheme(s)
	o := make(map[string]runtime.Object)

	m := mockK8sClient{
		objects: o,
		scheme:  s,
	}

	return &m
}

// Create adds the object to the map
func (m *mockK8sClient) Create(ctx context.Context, obj runtime.Object, opts ...client.CreateOption) error {
	key, err := m.getKey(obj)
	if err != nil {
		return err
	}
	_, exists := m.objects[key.String()]
	if exists {
		return apierrors.NewAlreadyExists(schema.GroupResource{}, key.String())
	}

	m.objects[key.String()] = obj

	return nil
}

// Get an object from the map
func (m *mockK8sClient) Get(ctx context.Context, key client.ObjectKey, obj runtime.Object) error {
	storedObj, exists := m.objects[key.String()]
	if !exists {
		return apierrors.NewNotFound(schema.GroupResource{}, key.String())
	}

	gvk, err := apiutil.GVKForObject(obj, m.scheme)
	if err != nil {
		return err
	}

	typeAccessor, err := meta.TypeAccessor(storedObj)
	if err != nil {
		return err
	}
	typeAccessor.SetKind(gvk.Kind)
	typeAccessor.SetAPIVersion(gvk.GroupVersion().String())

	jsonBytes, err := json.Marshal(storedObj)
	if err != nil {
		return err
	}
	decoder := scheme.Codecs.UniversalDecoder()
	_, _, err = decoder.Decode(jsonBytes, nil, obj)
	return err

}

// Update an object in the map
func (m *mockK8sClient) Update(ctx context.Context, obj runtime.Object, opts ...client.UpdateOption) error {
	key, err := m.getKey(obj)
	if err != nil {
		return err
	}

	_, exists := m.objects[key.String()]
	if !exists {
		return apierrors.NewNotFound(schema.GroupResource{}, key.String())
	}

	m.objects[key.String()] = obj

	return nil
}

// Delete an object from the map
func (m *mockK8sClient) Delete(ctx context.Context, obj runtime.Object, opts ...client.DeleteOption) error {
	key, err := m.getKey(obj)
	if err != nil {
		return err
	}

	_, exists := m.objects[key.String()]
	if !exists {
		return apierrors.NewNotFound(schema.GroupResource{}, key.String())
	}

	delete(m.objects, key.String())

	return nil
}

// Patch is not used but is here to satisfy the interface implementation
func (m *mockK8sClient) Patch(ctx context.Context, obj runtime.Object, patch client.Patch, opts ...client.PatchOption) error {
	return fmt.Errorf("Patch not implemented")
}

// Status returns a status writer, which we can reuse the mock client for
func (m *mockK8sClient) Status() client.StatusWriter {
	return m
}

func (m *mockK8sClient) getKey(obj runtime.Object) (types.NamespacedName, error) {
	accessor, err := meta.Accessor(obj)
	if err != nil {
		return types.NamespacedName{
			Namespace: "",
			Name:      "",
		}, err
	}

	return types.NamespacedName{
		Namespace: accessor.GetNamespace(),
		Name:      accessor.GetName(),
	}, nil
}

func getGVRFromObject(obj runtime.Object, scheme *runtime.Scheme) (schema.GroupVersionResource, error) {
	gvk, err := apiutil.GVKForObject(obj, scheme)
	if err != nil {
		return schema.GroupVersionResource{}, err
	}
	gvr, _ := meta.UnsafeGuessKindToResource(gvk)
	return gvr, nil
}

type mockDNCClient struct {
}
