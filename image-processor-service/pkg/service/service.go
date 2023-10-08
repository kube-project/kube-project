package service

import (
	"context"
	"fmt"

	"github.com/rs/zerolog"
	"golang.org/x/sync/errgroup"

	"github.com/kube-project/image-processor-service/pkg/providers"
)

// Dependencies are providers which this service operates with.
type Dependencies struct {
	Consumer  providers.ConsumerProvider
	Processor providers.ProcessorProvider
	Logger    zerolog.Logger
}

// New creates a new service with all of its dependencies and configurations.
func New(deps Dependencies) *ImageProcessor {
	return &ImageProcessor{
		deps: deps,
	}
}

// ImageProcessor represents the service object of the receiver.
type ImageProcessor struct {
	deps Dependencies
}

// Run starts the service.
func (s *ImageProcessor) Run(ctx context.Context) error {
	s.deps.Logger.Info().Msg("Starting service...")

	// Create the channel on which the consumer and the processor can communicate.
	// This should be buffered.
	mediator := make(chan int, 1)

	g, ctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		if err := s.deps.Consumer.Consume(mediator); err != nil {
			return fmt.Errorf("failed to start consumer: %w", err)
		}

		return nil
	})
	g.Go(func() error {
		if err := s.deps.Processor.ProcessImages(ctx, mediator); err != nil {
			return fmt.Errorf("failed to start process images: %w", err)
		}

		return nil
	})

	if err := g.Wait(); err != nil {
		return fmt.Errorf("failed to wait for consumer and process image: %w", err)
	}

	return nil
}
