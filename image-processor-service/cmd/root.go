package main

import (
	"context"
	"flag"
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/kube-project/image-processor-service/pkg/providers/circuitbreaker"
	"github.com/kube-project/image-processor-service/pkg/providers/consumer"
	"github.com/kube-project/image-processor-service/pkg/providers/processor"
	"github.com/kube-project/image-processor-service/pkg/providers/storage"
	"github.com/kube-project/image-processor-service/pkg/service"
)

var (
	rootArgs struct {
		consumerConfig  consumer.Config
		processorConfig processor.Config
		dbConfig        storage.Config
	}
)

func init() {
	flag.StringVar(&rootArgs.dbConfig.Hostname, "db-host", "localhost", "--db-host=localhost")
	flag.StringVar(&rootArgs.dbConfig.UsernamePassword, "db-username-password", "root:password123", "--db-username-password=root:password123")
	flag.StringVar(&rootArgs.dbConfig.Dbname, "db-dbname", "kube", "--db-dbname=kube")
	flag.StringVar(&rootArgs.dbConfig.Port, "db-port", "3306", "--db-port=3306")
	flag.StringVar(&rootArgs.processorConfig.FaceRecognitionAddress, "face-recognition-service-address", "localhost:50051", "--face-recognition-service-address=localhost:50051")
	flag.StringVar(&rootArgs.consumerConfig.NsqAddress, "nsq-lookup-address", "127.0.0.1:4161", "--nsq-lookup-address=127.0.0.1:4161")
	flag.Parse()
}

func main() {
	// Wire up the service and its dependencies.
	logger := log.Output(zerolog.ConsoleWriter{Out: os.Stdout})
	cb := circuitbreaker.NewCircuitBreaker(logger)
	config := rootArgs.dbConfig
	config.Logger = logger
	storer := storage.NewMySQLStorage(config)
	proc, err := processor.NewProcessorProvider(rootArgs.processorConfig, processor.Dependencies{
		CircuitBreaker: cb,
		Logger:         logger,
		Storer:         storer,
	})
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to initiate the processor")
	}

	cons := consumer.NewConsumer(rootArgs.consumerConfig, consumer.Dependencies{Logger: logger})

	srvc := service.New(service.Dependencies{
		Processor: proc,
		Consumer:  cons,
		Logger:    logger,
	})

	if err := srvc.Run(context.Background()); err != nil {
		logger.Fatal().Err(err).Msg("Failed to run")
	}
}
