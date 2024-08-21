package util

import (
	"fmt"
	"strings"
)

func Contains(options []string, target string) bool {
	for _, t := range options {
		if t == target {
			return true
		}
	}

	return false
}

func traverseUnstructured(path, breadcrumbs string, unstructured map[string]any) (any, error) {
	pathParts := strings.Split(path, ".")

	next, ok := unstructured[pathParts[0]]
	if !ok {
		return struct{}{}, fmt.Errorf("unknown key %s at path %s", path, breadcrumbs)
	}

	if len(pathParts) == 1 {
		return next, nil
	}

	nextAsUnstructured, ok := next.(map[string]any)
	if !ok {
		return struct{}{}, fmt.Errorf("more traversal required, but value at %s is not map[string]any", breadcrumbs+"."+pathParts[0])
	}

	return traverseUnstructured(strings.Join(pathParts[1:], "."), breadcrumbs+"."+pathParts[0], nextAsUnstructured)
}

func TraverseUnstructured(path string, unstructured map[string]any) (any, error) {
	return traverseUnstructured(path, "", unstructured)
}
