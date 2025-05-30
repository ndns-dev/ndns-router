package interfaces

type Utils interface {
	// Path related
	IsInternalPath(path string) bool

	// Logging related
	Info(msg string)
	Infof(format string, args ...any)
	Error(msg string)
	Errorf(format string, args ...any)
	Warn(msg string)
	Warnf(format string, args ...any)
	Debug(msg string)
	Debugf(format string, args ...any)
	Fatal(msg string)
	Fatalf(format string, args ...any)

	// Generate related
	GenerateRequestId() string
	NextRoundRobinIndex(length int) int

	// Calculate related
	RandomFloat64() float64
	RandomInt(max int) int

	// RoundRobin related
	RoundRobin(items []string) string
	RoundRobinIndex(items []string) int
}
