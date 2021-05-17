package filesystem

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"sync"
)

type SecretProvider struct {
	SecretFilenames map[string]string
	mutex           sync.Mutex
}

func (s *SecretProvider) GetLatestSecret(_ context.Context, name string) (string, error) {
	s.mutex.Lock()
	filename, found := s.SecretFilenames[name]
	s.mutex.Unlock()

	if !found {
		return "", errors.New("secret not found")
	}

	bytes, err := ioutil.ReadFile(filename)
	if err != nil {
		return "", fmt.Errorf("unable to get secret: %w", err)
	}

	return string(bytes), nil
}
