// Copyright 2020 Microsoft. All rights reserved.

package kubernetes

func removeStringFromSlice(strings []string, remove string) ([]string, bool) {
	ret := []string{}
	deletedSomething := false
	for _, s := range strings {
		if s != remove {
			ret = append(ret, s)
		} else {
			deletedSomething = true
		}
	}
	return ret, deletedSomething
}

func containsString(strings []string, s string) bool {
	for _, t := range strings {
		if t == s {
			return true
		}
	}

	return false
}

func appendStringIfNotPresent(strings []string, s string) []string {
	for _, t := range strings {
		if t == s {
			return strings
		}
	}

	return append(strings, s)
}
