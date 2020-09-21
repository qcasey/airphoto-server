package comment

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/qcasey/airphoto-server/internal/database"
	"github.com/qcasey/nskeyedarchiver"
	"github.com/qcasey/plist"
	"github.com/rs/zerolog/log"
)

// Comment corresponds to each row in the 'chat' table, along with a lock for the Messages
type Comment struct {
	AssetGUID string    `json:"AssetGUID"`
	GUID      string    `json:"GUID"`
	Date      time.Time `json:"Date"`
	IsCaption bool      `json:"IsCaption"`
	IsMine    bool      `json:"IsMine"`
	IsLike    bool      `json:"IsLike"`
	Name      string    `json:"Name"`
	Email     string    `json:"Email"`
	Content   string    `json:"Content"`
}

var determinedName string

func parseCommentRows(rows *sql.Rows) map[string]*Comment {
	out := make(map[string]*Comment, 0)
	var (
		tempTime float64
		tempObj  []byte
		err      error
	)

	for rows != nil && rows.Next() {
		c := Comment{}

		rows.Scan(&c.AssetGUID, &c.GUID, &tempTime, &c.IsCaption, &c.IsMine, &tempObj)

		if c.Date, err = nskeyedarchiver.NSDateToTime(tempTime); err != nil {
			log.Error().Msg(err.Error())
			continue
		}
		/*
			if err := c.parseCommentObj(&tempObj); err != nil {
				log.Error().Msg(err.Error())
				continue
			}*/

		// Set the user's name based on the IsMine bool
		if determinedName == "" && c.IsMine && c.Name != "" {
			determinedName = c.Name
		}

		// Append to output list
		out[c.Date.Format(time.RFC3339Nano)] = &c
	}

	return out
}

func (comment *Comment) ParseCommentObj(obj *[]byte) error {
	if comment == nil || obj == nil {
		return fmt.Errorf("Empty comment")
	}
	var err error

	// Extract values from NSArchive
	if comment.Name, err = plist.GetValue(obj, "fullName"); err != nil {
		return err
	}
	if comment.Email, err = plist.GetValue(obj, "email"); err != nil {
		return err
	}
	if comment.IsLike, err = plist.IsLike(obj); err != nil {
		return err
	}
	if comment.IsLike {
		return nil
	}
	if comment.Content, err = plist.GetValue(obj, "content"); err != nil {
		return err
	}
	return nil
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
