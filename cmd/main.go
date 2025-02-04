package main

import (
	"log"
	"net/http"

	"github.com/CodeWithKrushnal/ChainBank/internal/app"
	"github.com/CodeWithKrushnal/ChainBank/internal/config"
)

func main() {
	// Config Setup
	postgresDB, ethClient := config.InitConfig()
	defer config.ReleaseConfig(postgresDB)

	deps := app.NewDependencies(postgresDB, ethClient)

	router := app.SetupRoutes(deps)
	log.Println("Server started on port 8080")
	log.Fatal(http.ListenAndServe(":8080", router))
}
