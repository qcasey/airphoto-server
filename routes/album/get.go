package album

import (
	"encoding/json"
	"net/http"
	"sort"

	"github.com/gorilla/mux"
	"github.com/qcasey/airphoto/pkg/album"
	"github.com/qcasey/airphoto/pkg/asset"
	"github.com/qcasey/airphoto/server"
)

func Get(srv server.Server) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		srv.Mutex.RLock()
		defer srv.Mutex.RUnlock()

		params := mux.Vars(r)
		for _, a := range srv.Albums {
			if a.GUID == params["guid"] {
				// Sort assets
				assets := make(asset.List, len(a.Assets))
				for _, asset := range a.Assets {
					assets = append(assets, *asset)
				}
				sort.Sort(assets)
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(assets)
				return
			}
		}

		w.WriteHeader(http.StatusNotFound)
	}
}

func GetAll(srv server.Server) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		srv.Mutex.RLock()
		defer srv.Mutex.RUnlock()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(srv.Albums)
	}
}

func GetList(srv server.Server) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		srv.Mutex.RLock()
		defer srv.Mutex.RUnlock()

		assetlessAlbums := make(album.List, 0, len(srv.Albums))
		for _, a := range srv.Albums {
			a2 := *a
			a2.Assets = nil
			assetlessAlbums = append(assetlessAlbums, a2)
		}
		sort.Sort(assetlessAlbums)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(srv.Albums)
	}
}

func handleAlbumGetList(w http.ResponseWriter, r *http.Request) {

}
