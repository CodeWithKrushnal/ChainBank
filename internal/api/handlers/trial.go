package handlers

import (
	"log"
	"net/http"
)

func TrialHandler(w http.ResponseWriter, r *http.Request) {
	log.Println(r.Context())
	w.Write([]byte("Received Successfully"))
}
