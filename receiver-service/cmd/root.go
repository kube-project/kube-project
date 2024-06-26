package main

import (
	"context"
	"flag"
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/kube-project/receiver-service/pkg/providers/images"
	"github.com/kube-project/receiver-service/pkg/providers/sender"
	"github.com/kube-project/receiver-service/pkg/service"
)

var (
	rootArgs struct {
		imgConfig    images.Config
		senderConfig sender.Config
	}
)

func init() {
	flag.StringVar(&rootArgs.imgConfig.Hostname, "db-host", "localhost", "--db-host=localhost")
	flag.StringVar(&rootArgs.imgConfig.UsernamePassword, "db-username-password", "root:password123", "--db-username-password=root:password123")
	flag.StringVar(&rootArgs.imgConfig.Dbname, "db-dbname", "kube", "--db-dbname=kube")
	flag.StringVar(&rootArgs.imgConfig.Port, "db-port", "3306", "--db-port=3306")
	flag.StringVar(&rootArgs.senderConfig.Address, "producer-address", "127.0.0.1:4150", "--producer-address=127.0.0.1:4150")
	flag.Parse()
}

func main() {
	logger := log.Output(zerolog.ConsoleWriter{Out: os.Stdout})
	rootArgs.imgConfig.Logger = logger
	imgProvider := images.NewImageProvider(rootArgs.imgConfig)
	rootArgs.senderConfig.Logger = logger
	senderProvider := sender.NewNSQSender(rootArgs.senderConfig)

	srvc := service.New(service.Dependencies{
		ImageProvider: imgProvider,
		SendProvider:  senderProvider,
		Logger:        logger,
	})

	if err := srvc.Run(context.Background()); err != nil {
		logger.Fatal().Err(err).Msg("Failed to start service")
	}
}
