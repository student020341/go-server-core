package TWTServer

import (
	"net/http"
	"strings"
)

// remove empty strings from slice
func fixPath(path []string) []string {
	var tmp []string
	for _, value := range path {
		if value != "" {
			tmp = append(tmp, value)
		}
	}
	return tmp
}

func PathFromRequest(r *http.Request) []string {
	return PathFromString(r.URL.Path, "/")
}

func PathFromString(url string, del string) []string {
	return fixPath(strings.Split(url, del))
}
