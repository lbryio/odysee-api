package status

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/lbryio/lbrytv/app/router"

	ljsonrpc "github.com/lbryio/lbry.go/v2/extras/jsonrpc"
)

var PlayerServers = []string{
	"https://player1.lbry.tv",
	"https://player2.lbry.tv",
	"https://player3.lbry.tv",
}

const (
	StatusOK       = "ok"
	StatusNotReady = "not_ready"
	StatusOffline  = "offline"
	StatusFailing  = "failing"
)

type ServerItem struct {
	Address string `json:"address"`
	Status  string `json:"status"`
	Error   string `json:"error,omitempty"`
}
type ServerList []ServerItem

func GetStatus(w http.ResponseWriter, req *http.Request) {
	services := map[string]ServerList{
		"lbrynet": ServerList{},
		"player":  ServerList{},
	}
	response := map[string]interface{}{
		"timestamp":     fmt.Sprintf("%v", time.Now().UTC()),
		"services":      services,
		"general_state": StatusOK,
	}
	respStatus := http.StatusOK
	failureDetected := false

	router := router.NewDefault()
	sdks := router.GetSDKServerList()
	for _, s := range sdks {
		c := ljsonrpc.NewClient(s.Address)
		status, err := c.Status()
		srv := ServerItem{Address: s.Address, Status: StatusOK}
		if err != nil {
			srv.Error = fmt.Sprintf("%v", err)
			srv.Status = StatusOffline
			respStatus = http.StatusServiceUnavailable
			failureDetected = true
		} else if !status.StartupStatus.Wallet {
			srv.Status = StatusNotReady
			respStatus = http.StatusServiceUnavailable
			failureDetected = true
		}
		services["lbrynet"] = append(services["lbrynet"], srv)
	}

	for _, ps := range PlayerServers {
		r, err := http.Get(ps)
		srv := ServerItem{Address: ps, Status: StatusOK}
		if err != nil {
			srv.Error = fmt.Sprintf("%v", err)
			srv.Status = StatusOffline
			respStatus = http.StatusServiceUnavailable
			failureDetected = true
		} else if r.StatusCode != http.StatusNotFound {
			srv.Status = StatusNotReady
			srv.Error = fmt.Sprintf("http status %v", r.StatusCode)
			respStatus = http.StatusServiceUnavailable
			failureDetected = true
		}
		services["player"] = append(services["player"], srv)
	}
	if failureDetected {
		response["general_state"] = StatusFailing
	}

	w.Header().Add("content-type", "application/json; charset=utf-8")
	w.WriteHeader(respStatus)
	respByte, _ := json.MarshalIndent(&response, "", "  ")
	w.Write(respByte)
}