package utils

// UtilsImpl implements the interfaces.Utils interface
type UtilsImpl struct {
	*Log
	*Path
	*Calculate
	*Generate
}

// NewUtils creates a new instance of UtilsImpl
func NewUtils(internalPaths map[string]bool) *UtilsImpl {
	return &UtilsImpl{
		Log:       NewLog(),
		Path:      NewPath(internalPaths),
		Calculate: NewCalculate(),
		Generate:  NewGenerate(),
	}
}
