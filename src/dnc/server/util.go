package server

import (
	"net/url"
	"strconv"
)

func extractBoolQueryParam(u *url.URL, param string) (bool, error) {
	v := u.Query().Get(param)
	if v == "" {
		return false, nil
	}
	return strconv.ParseBool(v)
}
