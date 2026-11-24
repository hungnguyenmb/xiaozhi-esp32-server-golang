//go:build !cgo || (!linux && !darwin)

package denoise

import "fmt"

type nativeStream struct{}

func newNative() (*nativeStream, error) {
	return nil, fmt.Errorf("RNNoise needs cgo on Linux or macOS")
}
func (*nativeStream) process([]float32) ([]float32, error) {
	return nil, fmt.Errorf("RNNoise unavailable")
}
func (*nativeStream) close() {}

func (*nativeStream) latencySamples() int { return 0 }
