package providers

import "context"

// ProcessorProvider defines a set of functions which a processor needs.
type ProcessorProvider interface {
	ProcessImages(ctx context.Context, in chan int) error
}
