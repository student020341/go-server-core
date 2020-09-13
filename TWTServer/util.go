package TWTServer

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
