// Package handler provides shared utility functions for job service handlers.
package handler

import "strings"

// parsePath splits a URL path into components.
func parsePath(path string) []string {
	return strings.Split(strings.Trim(path, "/"), "/")
}

// nullIfEmpty returns nil if string is empty, otherwise returns the string.
func nullIfEmpty(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}
