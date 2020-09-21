package server

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	// For reading Apple's cloudphotodb
	_ "github.com/mattn/go-sqlite3"

	"github.com/gorilla/mux"
	"github.com/qcasey/airphoto-server/internal/database"
	"github.com/qcasey/airphoto-server/pkg/album"
	"github.com/qcasey/airphoto-server/server/config"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

// Server holds global state of the iMessage server
type Server struct {
	router *mux.Router // the api service's route collection
	*viper.Viper

	// Main map and submaps of parsed album data
	Albums []*album.Album

	// Device Tokens for firebase messaging
	DeviceTokens []string
	useFirebase  bool

	Started bool

	// DeterminedName is the user's name derived from comment's "IsMine" bool.
	DeterminedName string
	Mutex          sync.RWMutex
}

func New() (*Server, error) {
	r := &Server{
		Albums:  make([]*album.Album, 0),
		Started: false,
		Viper:   config.Read(),
	}

	return r, nil
}

func (s *Server) Start(binder func(s *Server, r *mux.Router)) {
	err := database.Open()
	if err != nil {
		log.Fatal().Err(err).Msg("Could not setup database")
	}

	s.router = mux.NewRouter().StrictSlash(true)
	database.File = s.Viper.GetString("db")
	go s.infiniteReader(time.Duration(s.Viper.GetInt("recheckInterval")) * time.Millisecond)
	binder(s, s.router)

	l, err := net.Listen("tcp", fmt.Sprintf(":%d", s.Viper.GetInt("port")))
	if err != nil {
		log.Fatal().Err(err).Msg("Could not create listener")
	}

	if err := http.Serve(l, s.router); errors.Is(err, http.ErrServerClosed) {
		log.Warn().Err(err).Msg("Web server has shut down")
	} else {
		log.Fatal().Err(err).Msg("Web server has shut down unexpectedly")
	}
}

func checkForNotificationsToSend(newAlbums []*album.Album) {
	return
	/*
		// Compare asset against current set for notifications or early termination
		oldAsset, assetExists := server.Albums[asset.AlbumGUID].Assets[asset.GUID]

		if assetExists {
			asset = oldAsset
			log.Info().Msgf("Asset %s exists, skipping.", asset.GUID)
		} else {
			newAssetAuthors[asset.Author]++
		}

		// Send comments notification
		if newCommentCount > 0 {
			sendNotification(server.Albums[albumGUID].Name, fmt.Sprintf("%d new comments", newCommentCount))
		}

		// Send asset notification
		if len(newAssetAuthors) > 0 {
			log.Info().Msg("New assets found, sending notifcations")
			for author, count := range newAssetAuthors {
				// Don't send notification about our own posted assets
				if author == server.DeterminedName {
					continue
				}

				// Asset is new and not mine, send device a notification
				notificationText := fmt.Sprintf("%s posted a new photo.", author)
				if count > 1 {
					notificationText = fmt.Sprintf("%s posted %d new photos.", author, count)
				}

				sendNotification(
					server.Albums[albumGUID].Name,
					notificationText,
				)
			}
		}*/
}

func (srv *Server) infiniteReader(interval time.Duration) {
	for {
		// Do initial startup
		if !srv.Started {
			log.Info().Msg("Building map of assets, this may take a while...")

			album.GetAlbums(false)
			srv.Started = true
			database.HasBeenModified() // set modified time
			continue
		}

		if database.HasBeenModified() {
			log.Info().Msg("DB file has been modified. Refreshing albums...")
			newAlbums, err := album.GetAlbums(true)
			if err != nil {
				log.Error().Err(err).Msg("Failed to refresh albums")
			}

			checkForNotificationsToSend(newAlbums)

			srv.Mutex.Lock()
			srv.Albums = newAlbums
			srv.Mutex.Unlock()
		}
		time.Sleep(interval)
	}
}
