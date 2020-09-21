package main

import (
	"os"
	"time"

	"github.com/qcasey/airphoto/server"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func init() {
	defaultZone, err := time.LoadLocation("America/Los_Angeles")
	if err != nil {
		log.Panic().Msg(err.Error())
	}

	// Configure logging
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	zerolog.TimestampFunc = func() time.Time {
		return time.Now().In(defaultZone)
	}

	output := zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: "Mon Jan 2 15:04:05"}
	log.Logger = zerolog.New(output).With().Caller().Timestamp().Logger()
}

func main() {
	srv, err := server.New()
	if err != nil {
		log.Fatal().Err(err).Msg("Could not create new server")
	}

	srv.Start(bindRoutes)
}
