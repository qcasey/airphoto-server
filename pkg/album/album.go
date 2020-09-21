package album

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/qcasey/airphoto-server/internal/database"
	"github.com/qcasey/airphoto-server/pkg/asset"
	"github.com/rs/zerolog/log"
)

// Album holds a list of assets and metadata for that album
type Album struct {
	GUID          string                  `json:"GUID"`
	Name          string                  `json:"Name"`
	URL           string                  `json:"URL"`
	LastPhotoDate time.Time               `json:"LastPhotoDate"`
	CoverPhoto    string                  `json:"CoverPhoto"`
	Assets        map[string]*asset.Asset `json:"Assets"`
}

// List for sorting albums
type List []Album

// Len is part of sort.Interface.
func (a List) Len() int {
	return len(a)
}

// Swap is part of sort.Interface.
func (a List) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

// Less is part of sort.Interface. We use count as the value to sort by
func (a List) Less(i, j int) bool {
	return a[i].LastPhotoDate.After(a[j].LastPhotoDate)
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

func GetAlbums(isRefresh bool) ([]*Album, error) {
	rows, err := database.Query("SELECT GUID, name, url FROM Albums")
	if err != nil {
		return nil, err
	}
	newAlbums := parseAlbumRows(rows)

	for _, Album := range newAlbums {
		log.Info().Msg(fmt.Sprintf("Parsing album %s (%s)", Album.Name, Album.GUID))

		var mostRecentAsset *asset.Asset
		Album.Assets, mostRecentAsset = asset.GetAssets(Album.GUID, isRefresh)
		Album.LastPhotoDate, Album.CoverPhoto = mostRecentAsset.SortingDate, mostRecentAsset.Path
	}

	log.Info().Msg(fmt.Sprintf("Parsed %d albums.", len(newAlbums)))
	return newAlbums, nil
}
