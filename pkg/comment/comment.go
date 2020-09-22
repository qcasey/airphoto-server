package comment

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/qcasey/airphoto-server/internal/database"
	"github.com/qcasey/nskeyedarchiver"
	"github.com/rs/zerolog/log"
)

// Comment corresponds to each row in the 'chat' table, along with a lock for the Messages
type Comment struct {
	AssetGUID   string    `json:"AssetGUID"`
	GUID        string    `json:"GUID"`
	Date        time.Time `json:"Date" mapstructure:"timestamp"`
	IsCaption   bool      `json:"IsCaption"`
	IsMine      bool      `json:"IsMine"`
	IsLike      bool      `json:"IsLike"`
	AuthorID    string    `json:"AuthorID" mapstructure:"personID"`
	AuthorName  string    `json:"Name" mapstructure:"fullName"`
	AuthorEmail string    `json:"Email"`
	Content     string    `json:"Content"`
}

var determinedName string

func parseCommentRows(rows *sql.Rows) map[string]*Comment {
	out := make(map[string]*Comment, 0)
	var (
		tempTime      float64
		embeddedPlist []byte
		err           error
	)

	for rows != nil && rows.Next() {
		c := Comment{}

		rows.Scan(&c.AssetGUID, &c.GUID, &tempTime, &c.IsCaption, &c.IsMine, &embeddedPlist)

		if c.Date, err = nskeyedarchiver.NSDateToTime(tempTime); err != nil {
			log.Error().Msg(err.Error())
			continue
		}

		plistData, err := nskeyedarchiver.Unarchive(embeddedPlist)
		if err != nil {
			fmt.Println("Error decoding plist:", err)
			continue
		}
		plistMap := plistData[0].(map[string]interface{})
		err = mapstructure.Decode(plistMap, &c)
		if err != nil {
			fmt.Println("Error mapping plist:", err)
			continue
		}

		// Set the user's name based on the IsMine bool
		if determinedName == "" && c.IsMine && c.AuthorName != "" {
			determinedName = c.AuthorName
		}

		// Append to output list
		out[c.Date.Format(time.RFC3339Nano)] = &c
	}

	return out
}

func GetComments(assetGUID string, oldComments *map[string]*Comment) map[string]*Comment {
	var excludeSlice strings.Builder
	if oldComments != nil && len(*oldComments) > 0 {
		log.Info().Msgf("Searching for refreshed comments, excluding %d existing ones", len(*oldComments))
		excludeSlice.WriteString(" AND Comments.GUID NOT IN (")
		isFirstComment := true
		for _, comment := range *oldComments {
			if !isFirstComment {
				excludeSlice.WriteString(",")
			}
			excludeSlice.WriteString(fmt.Sprintf("'%s'", comment.GUID))
			isFirstComment = false
		}
		excludeSlice.WriteString(")")
	}

	sql := fmt.Sprintf("SELECT AssetCollections.GUID, Comments.GUID, Comments.timestamp, Comments.isCaption, Comments.isMine, Comments.obj FROM Comments LEFT OUTER JOIN AssetCollections on AssetCollections.GUID = Comments.assetCollectionGUID WHERE AssetCollections.GUID = '%s'%s ORDER BY timestamp DESC", assetGUID, excludeSlice.String())
	//log.Info().Msg(sql)
	rows, err := database.Query(sql)
	if err != nil {
		log.Error().Msg(err.Error())
		return nil
	}
	defer rows.Close()

	return parseCommentRows(rows)
}
