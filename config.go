package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func parseProgramArguments() time.Duration {
	var (
		assetsPath     string
		dbPath         string
		intervalString string
	)
	flag.StringVar(&assetsPath, "assets", "", "Filepath to your iCloud Photos assets file (typically ~/Library/Containers/com.apple.cloudphotosd/Data/Library/Application Support/com.apple.cloudphotosd/services/com.apple.photo.icloud.sharedstreams/assets/)")
	flag.StringVar(&dbPath, "db", "", "Filepath to your iCloud db (typically ~/Library/Messages/chat.db)")
	flag.StringVar(&intervalString, "interval", "20000", "Interval in milliseconds to check for album updates")
	debug := flag.Bool("debug", false, "sets log level to debug")
	useFirebase := flag.Bool("firebase", false, "configures using Firebase API for notifications")
	flag.Parse()

	if *debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
		zerolog.CallerMarshalFunc = func(file string, line int) string {
			fileparts := strings.Split(file, "/")
			filename := strings.Replace(fileparts[len(fileparts)-1], ".go", "", -1)
			return filename + ":" + strconv.Itoa(line)
		}
	}

	server.useFirebase = *useFirebase
	if server.useFirebase {
		log.Info().Msg("Setting up Firebase notifications...")
		tokens, err := importDeviceTokens("./tokens")
		if err != nil {
			log.Warn().Msg(err.Error())
		} else {
			server.DeviceTokens = tokens
		}
	} else {
		log.Info().Msg("Not using Firebase notifications")
	}

	i, err := strconv.Atoi(intervalString)
	if err != nil {
		log.Panic().Msg(err.Error())
	}

	if dbPath == "" {
		exitWithMessage("--db Photos DB path is required.")
	}
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		exitWithMessage(fmt.Sprintf("Invalid photos DB. Are you sure this exists? %s", dbPath))
	}
	if assetsPath == "" {
		exitWithMessage("--assets Assets list is required.")

	}
	if _, err := os.Stat(assetsPath); os.IsNotExist(err) {
		exitWithMessage(fmt.Sprintf("Invalid assets directory. Are you sure this exists? %s", assetsPath))
	}

	log.Info().Msgf("Using DB at location %s", server.SQLiteFile)

	interval := time.Millisecond * time.Duration(i)
	server.AssetsPath = assetsPath
	server.SQLiteFile = dbPath
	log.Info().Msg(fmt.Sprintf("Checking DB every %f seconds", interval.Seconds()))
	return interval
}

func exitWithMessage(message string) {
	fmt.Println(message)
	fmt.Println("Type airphoto -h for a list of valid parameters and examples")
	os.Exit(2)
}
