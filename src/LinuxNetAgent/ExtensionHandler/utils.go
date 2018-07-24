package extensionhandler

import (
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/Azure/azure-container-networking/log"
)

func GetSettingsFiles(fileNames []string) []string {
	var settingFiles []string

	log.Printf("Filenames:%v", fileNames)
	var filterExp = regexp.MustCompile(`[0-9]+\.settings`)

	for _, file := range fileNames {
		if filterExp.MatchString(file) {
			settingFiles = append(settingFiles, file)
		}
	}

	return settingFiles
}

func FindLatestSettingNumber(fileNames []string) int {
	var settingNumbers []int

	for _, file := range fileNames {
		splitStr := strings.Split(file, ".")
		if len(splitStr) == 2 {
			num, _ := strconv.Atoi(splitStr[0])
			settingNumbers = append(settingNumbers, num)
		}
	}

	sort.Ints(settingNumbers)
	if len(settingNumbers) > 0 {
		return settingNumbers[len(settingNumbers)-1]
	}

	return 0
}
