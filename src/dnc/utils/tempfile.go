package utils

import (
	"fmt"
	"io/ioutil"
	"os"
)

// ToTempFile writes bytes to a temporary file, with an optional pattern, and returns the temporary file path
func ToTempFile(pattern string, bs []byte) (string, error) {
	tempFile, err := ioutil.TempFile(os.TempDir(), pattern)
	if err != nil {
		return "", fmt.Errorf("failed to create temp cert file: %v", err)
	}

	_, err = tempFile.Write(bs)
	if err != nil {
		return "", fmt.Errorf("failed to write to file: %v", err)
	}

	if err := tempFile.Close(); err != nil {
		return "", fmt.Errorf("failed to close file: %v", err)
	}

	return tempFile.Name(), nil
}
