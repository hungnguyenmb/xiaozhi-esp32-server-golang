//go:build !linux && !darwin

package capture

// The production free-space guard is supported on Linux and macOS.
func availableBytes(path string) (uint64, error) { return ^uint64(0), nil }
