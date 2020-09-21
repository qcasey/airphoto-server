package notification

import (
	"bufio"
	"fmt"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"github.com/qcasey/airphoto-server/server"
)

func stringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

// exportDeviceTokens writes the lines to the given file.
func exportDeviceTokens(path string, lines []string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	w := bufio.NewWriter(file)
	for _, line := range lines {
		fmt.Fprintln(w, line)
	}
	return w.Flush()
}

func Post(srv *server.Server) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		srv.Mutex.Lock()
		defer srv.Mutex.Unlock()

		params := mux.Vars(r)
		if params["token"] == "" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if stringInSlice(params["token"], srv.DeviceTokens) {
			w.WriteHeader(http.StatusNotModified)
			return
		}

		srv.DeviceTokens = append(srv.DeviceTokens, params["token"])
		exportDeviceTokens("./tokens", srv.DeviceTokens)
		w.WriteHeader(http.StatusOK)
	}
}
