package kubernetes

func removeStringFromSlice(strings []string, remove string) []string {
	ret := []string{}
	for _, s := range strings {
		if s != remove {
			ret = append(ret, s)
		}
	}
	return ret
}

func containsString(strings []string, s string) bool {
	for _, t := range strings {
		if t == s {
			return true
		}
	}

	return false
}
