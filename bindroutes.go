package main

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/qcasey/airphoto-server/routes/album"
	"github.com/qcasey/airphoto-server/routes/notification"
	"github.com/qcasey/airphoto-server/server"
)

func bindRoutes(srv *server.Server, r *mux.Router) {
	r.HandleFunc("/albums/{guid}", album.Get(srv)).Methods(http.MethodGet)
	r.HandleFunc("/albums", album.GetList(srv)).Methods(http.MethodGet)
	r.HandleFunc("/albums/all", album.GetAll(srv)).Methods(http.MethodGet)

	// Optionally handle firebase device tokens
	if srv.Viper.GetBool("useFirebase") {
		r.HandleFunc("/device/{token}", notification.Post(srv)).Methods("POST")
	}

	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		r.Write(w)
	}).Methods(http.MethodGet)
}
