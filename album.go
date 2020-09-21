package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"time"

	"github.com/gorilla/mux"
	"github.com/rs/zerolog/log"
)

// Album holds a list of assets and metadata for that album
type Album struct {
	GUID          string          `json:"GUID"`
	Name          string          `json:"Name"`
	URL           string          `json:"URL"`
	LastPhotoDate time.Time       `json:"LastPhotoDate"`
	CoverPhoto    string          `json:"CoverPhoto"`
	Assets        map[GUID]*Asset `json:"Assets"`
}

type albumSlice []Album

// Len is part of sort.Interface.
func (a albumSlice) Len() int {
	return len(a)
}

// Swap is part of sort.Interface.
func (a albumSlice) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

// Less is part of sort.Interface. We use count as the value to sort by
func (a albumSlice) Less(i, j int) bool {
	return a[i].LastPhotoDate.After(a[j].LastPhotoDate)
}

func handleAlbumGetAll(w http.ResponseWriter, r *http.Request) {
	server.lock.RLock()
	defer server.lock.RUnlock()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(server.Albums)
}

func handleAlbumGetList(w http.ResponseWriter, r *http.Request) {
	server.lock.RLock()
	defer server.lock.RUnlock()

	assetlessAlbums := make(albumSlice, 0, len(server.Albums))
	for _, a := range server.Albums {
		a2 := *a
		a2.Assets = nil
		assetlessAlbums = append(assetlessAlbums, a2)
	}
	sort.Sort(assetlessAlbums)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(server.Albums)
}

func handleAlbumGet(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)

	album, ok := server.Albums[params["guid"]]
	if !ok {
		writer.WriteHeader(http.StatusNotFound)
		return
	}

	// Sort assets
	assets := make(assetSlice, len(album.Assets))
	for _, asset := range album.Assets {
		assets = append(assets, *asset)
	}
	sort.Sort(assets)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(assets)
}

func parseAlbumRows(rows *sql.Rows) []*Album {
	var out []*Album
	if rows == nil {
		return out
	}
	defer rows.Close()
	for rows.Next() {
		c := Album{}
		rows.Scan(&c.GUID, &c.Name, &c.URL)
		out = append(out, &c)
	}
	return out
}

func getAlbums(isRefresh bool) {
	server.openDB()
	defer server.DB.Close()

	rows, err := query("SELECT GUID, name, url FROM Albums")
	if err != nil {
		log.Error().Msg(err.Error())
		return
	}
	newAlbums := parseAlbumRows(rows)

	albumCount := 0
	for _, Album := range newAlbums {
		log.Info().Msg(fmt.Sprintf("Parsing album %s (%s)", Album.Name, Album.GUID))

		var mostRecentAsset *Asset
		Album.Assets, mostRecentAsset = getAssets(Album.GUID, isRefresh)
		Album.LastPhotoDate, Album.CoverPhoto = mostRecentAsset.SortingDate, mostRecentAsset.Path

		server.lock.Lock()
		server.Albums[Album.GUID] = Album
		albumCount++
		server.lock.Unlock()
	}

	log.Info().Msg(fmt.Sprintf("Parsed %d albums.", albumCount))

	if isRefresh {
		log.Info().Msg("Refresh completed, closing DB file.")
	}
}
