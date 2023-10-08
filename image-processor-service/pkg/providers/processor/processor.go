package processor

import (
	"context"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/rs/zerolog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/kube-project/image-processor-service/facerecog"
	"github.com/kube-project/image-processor-service/pkg/models"
	"github.com/kube-project/image-processor-service/pkg/providers"
	"github.com/kube-project/image-processor-service/pkg/providers/circuitbreaker"
)

// Config needed for the processor.
type Config struct {
	FaceRecognitionAddress string
}

// Dependencies of the processor provider.
type Dependencies struct {
	Logger         zerolog.Logger
	CircuitBreaker circuitbreaker.CircuitBreaker
	Storer         providers.ImageStorer
}

// Processor defines a processor which uses a real database to store and process data.
type Processor struct {
	Dependencies
	Config
	IdentifyClient    facerecog.IdentifyClient
	HealthCheckClient facerecog.HealthCheckClient
}

// NewProcessorProvider creates a new processor provider with an active grpc connection.
func NewProcessorProvider(cfg Config, deps Dependencies) (providers.ProcessorProvider, error) {
	conn, err := grpc.Dial(cfg.FaceRecognitionAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("could not connect to grpc on: %s", cfg.FaceRecognitionAddress)
	}

	c := facerecog.NewIdentifyClient(conn)
	h := facerecog.NewHealthCheckClient(conn)
	return &Processor{
		Dependencies:      deps,
		Config:            cfg,
		IdentifyClient:    c,
		HealthCheckClient: h,
	}, nil
}

// updateImageWithFailedStatus updates a given image ID with failed status.
func (p *Processor) updateImageWithFailedStatus(imageID int) error {
	return p.Storer.UpdateImage(imageID, -1, models.FAILEDPROCESSING)
}

// updateImageWithPerson updates a record with the person's ID to which it belongs to.
func (p *Processor) updateImageWithPerson(personID, imageID int) error {
	return p.Storer.UpdateImage(imageID, personID, models.PROCESSED)
}

// ProcessImages takes a channel for input and waits on that channel for processable items.
// This channel must never be closed.
func (p *Processor) ProcessImages(ctx context.Context, in chan int) error {
	// Setup ping for the circuit breaker.
	p.CircuitBreaker.SetPingF(func() bool {
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()
		_, err := p.HealthCheckClient.HealthCheck(ctx, &facerecog.Empty{})
		return err != nil
	})

	// continuously get ids for image processing, block until something is received.
	for {
		select {
		case i := <-in:
			p.processImage(i)
		case <-ctx.Done():
			p.Logger.Debug().Msg("Process image context has been cancelled. Exiting.")
			return fmt.Errorf("context was cancelled")
		}
	}
}

// processImage will not retry processing a failed image or when the CircuitBreaker trips.
// It will just move on to the next image and mark that image failed in the Database.
// Further actions are taken on failed images once the Redeemer marks the image Pending again.
func (p *Processor) processImage(i int) {
	p.Logger.Info().Int("image-id", i).Msg("Processing image...")

	image, err := p.Storer.GetImage(i)
	if err != nil {
		p.Logger.Error().Err(err).Int("image-id", i).Msg("error while getting path for image")
		// log the error then continue
		return
	}
	if image.Status != models.PENDING {
		p.Logger.Debug().Int("image-id", i).Msg("image has already been processed")
		return
	}
	if err := p.Storer.UpdateImage(i, -1, models.PROCESSING); err != nil {
		p.Logger.Error().Err(err).Msg("Failed to mark the image as being processed...")
		return
	}
	p.CircuitBreaker.SetCallF(func() (*facerecog.IdentifyResponse, error) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()

		p.Logger.Debug().Str("path", image.Path).Msg("calling identify with image")
		r, err := p.IdentifyClient.Identify(ctx, &facerecog.IdentifyRequest{
			ImagePath: image.Path,
		})

		if err != nil {
			return nil, fmt.Errorf("failed to call face recognition service: %w", err)
		}

		return r, nil
	})
	r, err := p.CircuitBreaker.Call()
	if err != nil {
		if err := p.updateImageWithFailedStatus(i); err != nil {
			p.Logger.Error().Err(err).Msg("could not update image to failed status")
			return
		}

		p.Logger.Error().Err(err).Msg("image processing failed, updated image to failed status.")
		return
	}

	name := r.GetImageName()
	if name == "not_found" {
		if err := p.updateImageWithFailedStatus(i); err != nil {
			p.Logger.Error().Err(err).Msg("could not update image to failed status")
			return
		}

		p.Logger.Error().Msg("the person could not be identified")
		return
	}

	p.Logger.Info().Str("name", name).Msg("got name from face recog processor")
	person, err := p.Storer.GetPersonFromImage(name)
	if err != nil {
		if err := p.updateImageWithFailedStatus(i); err != nil {
			p.Logger.Error().Err(err).Msg("could not update image to failed status")
			return
		}

		p.Logger.Error().Err(err).Msg("could not retrieve person")
		return
	}

	p.Logger.Info().Str("person-name", person.Name).Msg("got person... updating record with person id")
	if err := p.updateImageWithPerson(person.ID, i); err != nil {
		p.Logger.Error().Err(err).Msg("warning: could not update image record")
		return
	}

	p.Logger.Info().Str("name", name).Msg("done")
}
