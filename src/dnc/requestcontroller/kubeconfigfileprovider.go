// Copyright 2020 Microsoft. All rights reserved.

package requestcontroller

import (
	"context"
	"io/ioutil"
)

type KubeconfigFromFileProvider struct {
	Filename string
}

const (
	fromFileOrchestratorType = "from-file"
)

var _ KubeconfigProvider = KubeconfigFromFileProvider{}

func (k KubeconfigFromFileProvider) GetBytesForAll(ctx context.Context) (map[string][]byte, error) {
	kubeconfigBytes, err := ioutil.ReadFile(k.Filename)
	if err != nil {
		return nil, err
	}

	return map[string][]byte{fromFileOrchestratorType: kubeconfigBytes}, nil
}
