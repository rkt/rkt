package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
)

type request struct {
	ContainerID string `json:"containerID"`
}

type errHandler func(http.ResponseWriter, *http.Request) *appError

type appError struct {
	Error   error
	Message string
	Code    int
}

func (fn errHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if e := fn(w, r); e != nil {
		log.Println("error:", e.Error)
		http.Error(w, e.Message, e.Code)
	}
}

func requestIP(w http.ResponseWriter, r *http.Request) *appError {
	networkName := r.URL.Path
	conf, err := NewNetworkConfig(networkName)
	if err != nil {
		return &appError{err, "", http.StatusNotFound}
	}

	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return &appError{err, "", http.StatusInternalServerError}
	}
	defer r.Body.Close()

	var req request
	err = json.Unmarshal(data, &req)
	if err != nil {
		return &appError{err, "", http.StatusInternalServerError}
	}

	allocator, err := NewIPAllocator(conf, store)
	if err != nil {
		return &appError{err, "", http.StatusInternalServerError}
	}
	allocation, err := allocator.Get(req.ContainerID)
	if err != nil {
		return &appError{err, "", http.StatusNoContent}
	}

	data, err = json.MarshalIndent(&allocation, "", "    ")
	if err != nil {
		return &appError{err, "", http.StatusInternalServerError}
	}
	log.Printf("allocated %s to %s", allocation.IP, req.ContainerID)
	w.Write(data)
	return nil
}
