package main

import (
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var server ServerType

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

	// Init Server
	server = ServerType{Started: false, Albums: make(map[string]*Album, 0)}
}

func main() {
	interval := parseProgramArguments()

	// Start router and DB readers
	go startDatabaseReader(interval)
	startRouter()
}

func startDatabaseReader(interval time.Duration) {
	for {
		// Do initial startup
		if !server.Started {
			log.Info().Msg("Building map of assets, this may take a while...")

			getAlbums(false)
			server.Started = true
			hasBeenModified() // set modified time
			continue
		}

		if hasBeenModified() {
			log.Info().Msg("DB file has been modified. Refreshing albums...")
			getAlbums(true)
		}
		time.Sleep(interval)
	}
}
