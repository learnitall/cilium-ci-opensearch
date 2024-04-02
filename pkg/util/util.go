package util

func Contains(options []string, target string) bool {
	for _, t := range options {
		if t == target {
			return true
		}
	}

	return false
}
