package config

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/filecoin-project/go-address"
)

type MinerAPI struct {
	Miner string  `json:"miner"`
	API   APIInfo `json:"api"`
}

func (dc *DynamicConfig) ReloadHandle(w http.ResponseWriter, r *http.Request) {
	log.Info("received reload config request")
	dc.reloadRequest <- struct{}{}
}

func (dc *DynamicConfig) AddMinerHandle(w http.ResponseWriter, r *http.Request) {
	log.Debugw("AddMinerHandle", "path", r.URL.Path)
	var minerAPI MinerAPI
	err := json.NewDecoder(r.Body).Decode(&minerAPI)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	maddr, err := address.NewFromString(minerAPI.Miner)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = dc.addMiner(maddr, minerAPI.API)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (dc *DynamicConfig) RemoveMinerHandle(w http.ResponseWriter, r *http.Request) {
	log.Debugw("RemoveMinerHandle", "path", r.URL.Path)

	id := strings.TrimPrefix(r.URL.Path, "/miner/remove/")
	maddr, err := address.NewFromString(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = dc.removeMiner(maddr)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (dc *DynamicConfig) ListMinerHandle(w http.ResponseWriter, r *http.Request) {
	log.Debugw("ListMinerHandle", "path", r.URL.Path)

	var miners []string
	for _, m := range dc.MinersList() {
		miners = append(miners, m.String())
	}

	data, err := json.Marshal(&miners)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Write(data)
}
