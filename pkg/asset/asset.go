package asset

import (
	"fmt"
	"mime"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/cheggaaa/pb"
	"github.com/mitchellh/mapstructure"
	"github.com/nozzle/throttler"
	"github.com/qcasey/airphoto-server/internal/database"
	"github.com/qcasey/airphoto-server/pkg/comment"
	"github.com/qcasey/nskeyedarchiver"
	"github.com/rs/zerolog/log"
)

// Asset corresponds to each row in the 'chat' table, along with a lock for the Messages
type Asset struct {
	GUID          string    `json:"GUID"`
	AlbumGUID     string    `json:"AlbumGUID"`
	Date          time.Time `json:"Date" mapstructure:"timestamp"`
	SortingDate   time.Time `json:"SortingDate"`
	Author        string    `json:"Author" mapstructure:"fullName"`
	AuthorID      string    `json:"AuthorID" mapstructure:"personID"`
	IsMine        bool      `json:"IsMine"`
	IsVideo       bool      `json:"IsVideo"`
	Filename      string    `json:"Filename"`
	Filetype      string    `json:"Filetype"`
	MIME          string    `json:"MIME"`
	LocalPath     string    `json:"LocalPath"`
	ThumbnailPath string    `json:"ThumbnailPath"`
	Path          string    `json:"Path"`
	Width         uint64    `json:"Width"`
	Height        uint64    `json:"Height"`
	//Date       float64             `json:"Date"`
	//BatchDate       float64             `json:"BatchDate"`
	Number float64 `json:"PhotoNumber" mapstructure:"photoNumber"`
	//LastCommentDate time.Time           `json:"LastCommentDate"`
	Comments map[string]*comment.Comment `json:"Comments"`

	PlistAssetData []plistAsset `mapstructure:"assets"`
	//Other    map[string]interface{}      `mapstructure:",remain"`
}

// These are for decoding mapstructures of unarchived plists
type plistAssetMetadata struct {
	MSAssetMetadataAssetType      string
	MSAssetMetadataAssetTypeFlags uint64
	MSAssetMetadataFileSize       uint64
	MSAssetMetadataPixelHeight    uint64
	MSAssetMetadataPixelWidth     uint64
}

type plistAsset struct {
	GUID                       string
	AssetCollectionGUID        string
	AssetDataAvailableOnServer bool
	FileHash                   []uint8
	MediaAssetType             uint64
	Metadata                   plistAssetMetadata
}

// List for sorting assets
type List []Asset

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
	return a[i].SortingDate.After(a[j].SortingDate)
}

func parseComments(asset *Asset, isRefresh bool) int {
	newCommentCount := 0

	// Skip comment parsing on old comments
	/*
		if isRefresh {
			if oldAsset, ok := server.Albums[asset.AlbumGUID].Assets[asset.GUID]; ok {
				newComments := comment.GetComments(string(asset.GUID), &oldAsset.Comments)
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
		} else {*/
	asset.Comments = comment.GetComments(string(asset.GUID), nil)
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
	//}

	return newCommentCount
}

// Get group Assets?
//sql := "SELECT DISTINCT chat.ROWID, chat.chat_identifier, chat.guid, chat.display_name FROM message LEFT OUTER JOIN chat ON chat.room_name = message.cache_roomnames LEFT OUTER JOIN handle ON handle.ROWID = message.handle_id WHERE message.is_from_me = 0 AND chat.service_name = 'iMessage' AND message.handle_id > 0 ORDER BY message.date DESC"

// For a different sqlite file
// "SELECT Z_PK, ZENTRY, ZASSETALBUMGUID, ZASSETGUID, ZASSETINFO FROM ZCLOUDFEEDENTRYASSET ORDER BY Z_PK DESC LIMIT 250"

// GetAssets returns all assets included within a specific album
func GetAssets(albumGUID string, isRefresh bool) (map[string]*Asset, *Asset) {
	var (
		mostRecentAsset *Asset
		newAssetCount   int
	)

	// Get count
	sqlCount := fmt.Sprintf("SELECT COUNT(*) FROM AssetCollections WHERE albumGUID = \"%s\"", albumGUID)
	rows, errCount := database.Query(sqlCount)
	if errCount != nil {
		log.Error().Msg(errCount.Error())
		return nil, nil
	}
	if rows.Next() {
		rows.Scan(&newAssetCount)
	}

	sql := fmt.Sprintf("SELECT albumGUID, GUID, batchDate, photoNumber, obj FROM AssetCollections WHERE albumGUID = \"%s\" ORDER BY batchDate DESC", albumGUID)
	rows, err := database.Query(sql)
	if err != nil {
		log.Error().Msg(err.Error())
		return nil, nil
	}

	defer rows.Close()

	assetMap := make(map[string]*Asset, 0) // for returning new assets
	var assetMutex sync.Mutex

	// For program counting
	start := time.Now()
	bar := pb.StartNew(newAssetCount)
	t := throttler.New(6, newAssetCount)

	for rows.Next() {
		asset := &Asset{Comments: make(map[string]*comment.Comment, 0)}
		var (
			appleTime     float64
			embeddedPlist []byte
		)
		rows.Scan(&asset.AlbumGUID, &asset.GUID, &appleTime, &asset.Number, &embeddedPlist)

		// Parse date before throwing away appleTime
		if parsedDate, err := nskeyedarchiver.NSDateToTime(appleTime); err == nil {
			asset.Date = parsedDate
		}

		go func(asset *Asset, embeddedPlist []byte) {

			plistData, err := nskeyedarchiver.Unarchive(embeddedPlist)
			if err != nil {
				fmt.Println("Error decoding plist:", err)
				return
			}
			plistMap := plistData[0].(map[string]interface{})
			err = mapstructure.Decode(plistMap, &asset)
			if err != nil {
				fmt.Println("Error mapping plist:", err)
				return
			}

			// Parse author
			asset.Filetype = strings.ToLower(filepath.Ext(asset.Filename))
			asset.MIME = mime.TypeByExtension(asset.Filetype)
			asset.IsVideo = asset.Filetype == ".mp4" || asset.Filetype == ".mov"

			// Graduate plist asset to main height/width data
			for _, a := range asset.PlistAssetData {
				if a.Metadata.MSAssetMetadataAssetType == "derivative" {
					asset.Height = a.Metadata.MSAssetMetadataPixelHeight
					asset.Width = a.Metadata.MSAssetMetadataPixelWidth
					break
				}
			}

			parseComments(asset, isRefresh)

			assetMutex.Lock()
			assetMap[asset.GUID] = asset
			assetMutex.Unlock()

			// Check date of this asset, update most recent
			if mostRecentAsset == nil || (!asset.IsVideo && asset.SortingDate.After(mostRecentAsset.SortingDate)) {
				mostRecentAsset = asset
			}

			// Mark as done
			bar.Increment()
			t.Done(nil)
		}(asset, embeddedPlist)

		t.Throttle()
	}

	bar.Finish()
	log.Info().Msg(fmt.Sprintf("(%f seconds) Parsed %d total assets from album %s.", time.Since(start).Seconds(), newAssetCount, albumGUID))

	return assetMap, mostRecentAsset
}
