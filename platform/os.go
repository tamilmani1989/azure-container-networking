package platform

import (
	"bufio"
	"fmt"
	"io"
	"os"
)

// ReadFileByLines reads file line by line and return array of lines.
func ReadFileByLines(filename string) ([]string, error) {
	var (
		lineStrArr []string
	)

	f, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("Error opening %s file error %v", filename, err)
	}

	defer f.Close()

	r := bufio.NewReader(f)

	for {
		lineStr, err := r.ReadString('\n')
		if err != nil {
			if err != io.EOF {
				return nil, fmt.Errorf("Error reading %s file error %v", filename, err)
			}

			lineStrArr = append(lineStrArr, lineStr)
			break
		}

		lineStrArr = append(lineStrArr, lineStr)
	}

	return lineStrArr, nil
}
