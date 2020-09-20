package main

import (
	"context"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"mime"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/cheggaaa/pb"
	"github.com/nozzle/throttler"
	"github.com/qcasey/plist"
	"github.com/rs/zerolog/log"
	ffprobe "gopkg.in/vansante/go-ffprobe.v2"
)

// Asset corresponds to each row in the 'chat' table, along with a lock for the Messages
type Asset struct {
	GUID        GUID      `json:"GUID"`
	AlbumGUID   string    `json:"AlbumGUID"`
	Date        time.Time `json:"Date"`
	SortingDate time.Time `json:"SortingDate"`
	Author      string    `json:"Author"`
	//IsMine      bool      `json:"IsMine"`
	IsVideo       bool   `json:"IsVideo"`
	Filename      string `json:"Filename"`
	Filetype      string `json:"Filetype"`
	MIME          string `json:"MIME"`
	LocalPath     string `json:"LocalPath"`
	ThumbnailPath string `json:"ThumbnailPath"`
	Path          string `json:"Path"`
	Width         int    `json:"Width"`
	Height        int    `json:"Height"`
	//Date       float64             `json:"Date"`
	//BatchDate       float64             `json:"BatchDate"`
	Number float64 `json:"PhotoNumber"`
	//LastCommentDate time.Time           `json:"LastCommentDate"`
	Comments map[string]*Comment `json:"Comments"`
	//obj             []byte
}

type assetSlice []Asset

// GUID represented by a string for asset maps
type GUID string

// Len is part of sort.Interface.
func (a assetSlice) Len() int {
	return len(a)
}

// Swap is part of sort.Interface.
func (a assetSlice) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

// Less is part of sort.Interface. We use count as the value to sort by
func (a assetSlice) Less(i, j int) bool {
	return a[i].SortingDate.After(a[j].SortingDate)
}

func (asset *Asset) parseFile(obj *[]byte) {
	var err error

	// Parse author
	if asset.Author, err = plist.GetValue(obj, "fullName"); err != nil {
		log.Error().Msg(err.Error())
	}

	// Parse filename and set path
	if asset.Filename, err = plist.GetValue(obj, "fileName"); err != nil {
		log.Error().Msg(err.Error())
		return
	}
	asset.LocalPath = filepath.Join(server.AssetsPath, fmt.Sprintf("%s/%s.poster.JPG", asset.AlbumGUID, asset.GUID))
	asset.Path = fmt.Sprintf("/file/%s/%s/%s", asset.AlbumGUID, asset.GUID, asset.Filename)
	asset.ThumbnailPath = asset.Path // this will be default for images, videos will overwrite
	asset.Filetype = strings.ToLower(filepath.Ext(asset.Filename))
	asset.MIME = mime.TypeByExtension(asset.Filetype)

	if !fileExists(asset.LocalPath) {
		if asset.Filetype == ".mp4" || asset.Filetype == ".mov" {
			//log.Warn().Msgf("Asset %s does not exist at %s, skipping file parsing", asset.GUID, asset.LocalPath)
		}
		return
	}

	if asset.Filetype == ".mp4" || asset.Filetype == ".mov" {
		ctx, cancelFn := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancelFn()

		data, err := ffprobe.ProbeURL(ctx, asset.LocalPath)
		if err != nil {
			log.Error().Msg(err.Error())
		}
		stream := data.FirstVideoStream()
		asset.Height = stream.Height
		asset.Width = stream.Width
		asset.ThumbnailPath = fmt.Sprintf("%s-thumbnail.jpg", asset.Path)
		asset.IsVideo = true

		// Save thumbnail to file
		SaveFrame(asset.Width, asset.Height, asset.LocalPath, asset.ThumbnailPath)

	} else {
		asset.IsVideo = false
		if reader, err := os.Open(asset.LocalPath); err == nil {
			defer reader.Close()
			im, _, err := image.DecodeConfig(reader)
			if err != nil {
				log.Error().Msgf("Error when decoding image %s.", asset.Path)
				log.Error().Msg(err.Error())
				return
			}
			asset.Height = im.Height
			asset.Width = im.Width
		} else {
			log.Error().Msg(err.Error())
		}
	}
}

func (asset *Asset) parseComments(isRefresh bool) int {
	newCommentCount := 0

	// Skip comment parsing on old comments
	if isRefresh {
		server.lock.Lock()
		if oldAsset, ok := server.Albums[asset.AlbumGUID].Assets[asset.GUID]; ok {
			newComments := getComments(string(asset.GUID), &oldAsset.Comments)
			newCommentCount = len(newComments)

			// insert old and new comments into asset
			for commentTime, comment := range oldAsset.Comments {
				asset.Comments[commentTime] = comment
			}
			for commentTime, comment := range newComments {
				asset.Comments[commentTime] = comment
			}

			if newCommentCount > 0 {
				asset.SortingDate = oldAsset.SortingDate
			}
		}
		server.lock.Unlock()
	} else {
		asset.Comments = getComments(string(asset.GUID), nil)
		newCommentCount = len(asset.Comments)

		// Determine sorting date
		// Default the last comment date (i.e. if there are no comments)
		asset.SortingDate = asset.Date

		// Parse over all comments
		for _, comment := range asset.Comments {
			if comment.Date.After(asset.SortingDate) {
				asset.SortingDate = comment.Date
			}
		}
	}

	return newCommentCount
}

// Get group Assets?
//sql := "SELECT DISTINCT chat.ROWID, chat.chat_identifier, chat.guid, chat.display_name FROM message LEFT OUTER JOIN chat ON chat.room_name = message.cache_roomnames LEFT OUTER JOIN handle ON handle.ROWID = message.handle_id WHERE message.is_from_me = 0 AND chat.service_name = 'iMessage' AND message.handle_id > 0 ORDER BY message.date DESC"

// For a different sqlite file
// "SELECT Z_PK, ZENTRY, ZASSETALBUMGUID, ZASSETGUID, ZASSETINFO FROM ZCLOUDFEEDENTRYASSET ORDER BY Z_PK DESC LIMIT 250"
func getAssets(albumGUID string, isRefresh bool) (map[GUID]*Asset, *Asset) {
	var (
		mostRecentAsset *Asset
		newAssetCount   int
	)

	// Get count
	sqlCount := fmt.Sprintf("SELECT COUNT(*) FROM AssetCollections WHERE albumGUID = \"%s\"", albumGUID)
	rows, errCount := query(sqlCount)
	if errCount != nil {
		log.Error().Msg(errCount.Error())
		return nil, nil
	}
	if rows.Next() {
		rows.Scan(&newAssetCount)
	}

	sql := fmt.Sprintf("SELECT albumGUID, GUID, batchDate, photoNumber, obj FROM AssetCollections WHERE albumGUID = \"%s\" ORDER BY batchDate DESC", albumGUID)
	rows, err := query(sql)
	if err != nil {
		log.Error().Msg(err.Error())
		return nil, nil
	}

	defer rows.Close()

	newAssetAuthors := make(map[string]int, 0) // for new asset notifications
	newCommentCount := 0
	assetMap := make(map[GUID]*Asset, 0) // for returning new assets

	// For program counting
	start := time.Now()
	bar := pb.StartNew(newAssetCount)
	t := throttler.New(6, newAssetCount)

	for rows.Next() {
		asset := &Asset{Comments: make(map[string]*Comment, 0)}
		var (
			appleTime float64
			tempObj   []byte
		)
		rows.Scan(&asset.AlbumGUID, &asset.GUID, &appleTime, &asset.Number, &tempObj)

		// Parse date before throwing away appleTime
		if parsedDate, err := ParseAppleTimestamp(appleTime); err == nil {
			asset.Date = parsedDate
		}

		go func(asset *Asset, tempObj []byte) {
			newCommentCount += asset.parseComments(isRefresh)
			assetExists := false

			// Compare asset against current set for notifications or early termination
			if isRefresh {
				server.lock.RLock()
				oldAsset, assetExists := server.Albums[asset.AlbumGUID].Assets[asset.GUID]

				if assetExists {
					asset = oldAsset
					log.Info().Msgf("Asset %s exists, skipping.", asset.GUID)
				} else {
					newAssetAuthors[asset.Author]++
				}
				server.lock.RUnlock()
			}

			if !assetExists {
				asset.parseFile(&tempObj)
			}

			// Add to server
			server.lock.Lock()
			defer server.lock.Unlock()
			assetMap[asset.GUID] = asset

			// Check date of this asset, update most recent
			if mostRecentAsset == nil || (!asset.IsVideo && asset.SortingDate.After(mostRecentAsset.SortingDate)) {
				mostRecentAsset = asset
			}

			// Mark as done
			bar.Increment()
			t.Done(nil)
		}(asset, tempObj)

		t.Throttle()
	}

	if isRefresh {
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
		}
	}

	bar.Finish()
	log.Info().Msg(fmt.Sprintf("(%f seconds) Parsed %d total assets from album %s.", time.Since(start).Seconds(), newAssetCount, albumGUID))

	return assetMap, mostRecentAsset
}
