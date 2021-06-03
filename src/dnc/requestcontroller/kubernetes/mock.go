// Copyright 2020 Microsoft. All rights reserved.

package kubernetes

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

// mockK8sClient is a very minimal mock that stores objects in a map with a key equal to the objects "namespace/name". This is fine for request
// controller because it only interacts with nodes (no namespace) and NNCs (namespaced). This should not be used by any other project, and thus
// is package-private.
type mockK8sClient struct {
	objectMaps map[schema.GroupVersionResource]map[string]runtime.Object
}

func newMockK8sClient() *mockK8sClient {
	o := make(map[schema.GroupVersionResource]map[string]runtime.Object)

	m := mockK8sClient{
		objectMaps: o,
	}

	return &m
}

// Create adds the object to the map
func (m *mockK8sClient) Create(ctx context.Context, obj runtime.Object, opts ...client.CreateOption) error {
	key, err := m.getKey(obj)
	if err != nil {
		return err
	}
	objects, err := m.getMapForObjectType(obj)
	if err != nil {
		return err
	}
	_, exists := objects[key.String()]
	if exists {
		return apierrors.NewAlreadyExists(schema.GroupResource{}, key.String())
	}

	objects[key.String()] = obj

	return nil
}

// Get an object from the map
func (m *mockK8sClient) Get(ctx context.Context, key client.ObjectKey, obj runtime.Object) error {
	objects, err := m.getMapForObjectType(obj)
	if err != nil {
		return err
	}

	storedObj, exists := objects[key.String()]
	if !exists {
		return apierrors.NewNotFound(schema.GroupResource{}, key.String())
	}

	gvk, err := apiutil.GVKForObject(obj, scheme)
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
	decoder := clientgoscheme.Codecs.UniversalDecoder()
	_, _, err = decoder.Decode(jsonBytes, nil, obj)
	return err

}

// Update an object in the map
func (m *mockK8sClient) Update(ctx context.Context, obj runtime.Object, opts ...client.UpdateOption) error {
	objects, err := m.getMapForObjectType(obj)
	if err != nil {
		return err
	}

	key, err := m.getKey(obj)
	if err != nil {
		return err
	}

	_, exists := objects[key.String()]
	if !exists {
		return apierrors.NewNotFound(schema.GroupResource{}, key.String())
	}

	// if deletion timestamp is non-zero and finalizers are removed, delete it
	accessor, err := meta.Accessor(obj)
	if err != nil {
		return err
	}

	if len(accessor.GetFinalizers()) == 0 && !accessor.GetDeletionTimestamp().IsZero() {
		delete(objects, key.String())
		return nil
	}

	objects[key.String()] = obj

	return nil
}

// Delete an object from the map
func (m *mockK8sClient) Delete(ctx context.Context, obj runtime.Object, opts ...client.DeleteOption) error {
	objects, err := m.getMapForObjectType(obj)
	if err != nil {
		return err
	}

	key, err := m.getKey(obj)
	if err != nil {
		return err
	}

	storedObj, exists := objects[key.String()]
	if !exists {
		return apierrors.NewNotFound(schema.GroupResource{}, key.String())
	}

	accessor, err := meta.Accessor(storedObj)
	if err != nil {
		return err
	}

	// if there are finalizers, just mark it for deletion but don't remove
	if len(accessor.GetFinalizers()) != 0 {
		accessor.SetDeletionTimestamp(&metav1.Time{time.Now()})
		return nil
	}

	delete(objects, key.String())

	return nil
}

// Patch is not used but is here to satisfy the interface implementation
func (m *mockK8sClient) Patch(ctx context.Context, obj runtime.Object, patch client.Patch, opts ...client.PatchOption) error {
	return fmt.Errorf("patch not implemented")
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

func (m *mockK8sClient) getGVRFromObject(obj runtime.Object) (schema.GroupVersionResource, error) {
	gvk, err := apiutil.GVKForObject(obj, scheme)
	if err != nil {
		return schema.GroupVersionResource{}, err
	}
	gvr, _ := meta.UnsafeGuessKindToResource(gvk)
	return gvr, nil
}

func (m *mockK8sClient) getMapForObjectType(obj runtime.Object) (map[string]runtime.Object, error) {
	gvr, err := m.getGVRFromObject(obj)
	if err != nil {
		return nil, err
	}

	if gvrMap, ok := m.objectMaps[gvr]; ok {
		return gvrMap, nil
	}

	m.objectMaps[gvr] = make(map[string]runtime.Object)
	return m.objectMaps[gvr], nil
}
