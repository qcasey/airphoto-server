package main

import (
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/qcasey/MDroid-Core/format/response"
	"github.com/rs/zerolog/log"
)

// These should be set in a UI somewhere
const (
	Port      = ":1459"
	AuthToken = "NOT_IN_USE_YET"
)

func startRouter() {
	log.Info().Msgf("Using assets at directory %s", server.AssetsPath)

	// Init router
	router := mux.NewRouter()

	router.HandleFunc("/albums", handleAlbumGetList).Methods("GET")
	router.HandleFunc("/albums/all", handleAlbumGetAll).Methods("GET")
	router.HandleFunc("/albums/{guid}", handleAlbumGet).Methods("GET")
	//router.HandleFunc("/assets", handleAssetGetAll).Methods("GET")

	// Optionally handle firebase device tokens
	if server.useFirebase {
		router.HandleFunc("/device/{token}", handleDeviceTokenPost).Methods("POST")
	}

	router.
		PathPrefix("/file/").
		Handler(http.StripPrefix("/file/", http.FileServer(http.Dir(server.AssetsPath))))

	//
	// Finally, welcome and meta routes
	//
	router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		res := response.JSONResponse{Output: "OK", OK: true}
		res.Write(&w, r)
	}).Methods("GET")

	log.Info().Msg("Starting server on port 1459")

	// Start the router in an endless loop
	for {
		err := http.ListenAndServe(Port, router)
		log.Error().Msg(err.Error())
		log.Error().Msg("Router failed! We messed up really bad to get this far. Restarting the router...")
		time.Sleep(time.Second * 10)
	}
}

// authMiddleware will match http bearer token again the one hardcoded in our config
func authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		reqToken := r.Header.Get("Authorization")
		splitToken := strings.Split(reqToken, "Bearer")
		if len(splitToken) != 2 || strings.TrimSpace(splitToken[1]) != AuthToken {
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte("403 - Invalid Auth Token!"))
		}

		// Call the next handler, which can be another middleware in the chain, or the final handler.
		next.ServeHTTP(w, r)
	})
}
