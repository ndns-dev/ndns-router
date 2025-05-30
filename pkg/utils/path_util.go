package utils

import "strings"

// Path handles path-related operations
type Path struct {
	internalPaths map[string]bool
}

// NewPath creates a new instance of Path
func NewPath(internalPaths map[string]bool) *Path {
	return &Path{
		internalPaths: internalPaths,
	}
}

// IsInternalPath checks if the given path is an internal management path
func (p *Path) IsInternalPath(path string) bool {
	for internalPathPrefix := range p.internalPaths {
		if strings.HasPrefix(path, internalPathPrefix) {
			return true
		}
	}
	return false
}
