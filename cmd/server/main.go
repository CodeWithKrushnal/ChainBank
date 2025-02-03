package main

import (
	"log"
	"net/http"

	"github.com/CodeWithKrushnal/ChainBank/internal/api"
	"github.com/CodeWithKrushnal/ChainBank/internal/api/config"
)

func main() {
	// Config Setup 
	config.InitConfig()
	defer config.ReleaseConfig()

	router := api.SetupRoutes()
	log.Println("Server started on port 8080")
	log.Fatal(http.ListenAndServe(":8080", router))
}
