package server

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/rs/zerolog/log"
)

type notificationContainer struct {
	Notifications []notification `json:"notifications"`
}
type notification struct {
	Tokens           []string         `json:"tokens"`
	Platform         int              `json:"platform"`
	Message          string           `json:"message"`
	Title            string           `json:"title"`
	NotificationData notificationData `json:"notification"`
}

type notificationData struct {
	Icon  string `json:"icon"`
	Color string `json:"color"`
}

// sendNotification will send title and message to all registered deviceTokens
func (s *Server) sendNotification(title string, message string) {
	if len(s.DeviceTokens) == 0 {
		log.Warn().Msg("No device tokens registered, not sending notification.")
		return
	}

	jsonStr, err := json.Marshal(notificationContainer{
		Notifications: []notification{
			notification{
				Tokens:   s.DeviceTokens,
				Platform: 2,
				Message:  message,
				Title:    title,
				NotificationData: notificationData{
					Icon:  "AirPhoto",
					Color: "#5EA5F5",
				},
			},
		},
	})
	if err != nil {
		log.Error().Msg(err.Error())
		return
	}
	log.Info().Msg(string(jsonStr))

	url := "https://notifications.airphoto.app"
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonStr))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Error().Msg(err.Error())
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := ioutil.ReadAll(resp.Body)
		log.Error().Msg(string(body))
	}
}

// importDeviceTokens reads the lines to a return slice.
func importDeviceTokens(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines, scanner.Err()
}
