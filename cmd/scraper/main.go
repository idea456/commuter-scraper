package main

import (
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"os"

	"github.com/idea456/commuter-scraper/internal/scraper"
	"github.com/idea456/commuter-scraper/internal/solver"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Fatal("Can't find or load .env file")
		os.Exit(1)
	}

	solverServer, err := solver.NewSolverServer()
	if err != nil {
		slog.Error(fmt.Sprintf("Can't initialise solver server: %v", err))
		os.Exit(1)
	}
	listingScraper := scraper.NewListingScraper(solverServer)

	listings, err := listingScraper.ScrapeListing()
	if err != nil {
		slog.Error(fmt.Sprintf("error in scraping listings: %v", err))
		os.Exit(1)
	}

	listingsB, err := json.Marshal(listings)
	if err != nil {
		slog.Error(fmt.Sprintf("error in marshalling listings to bytes: %v", err))
		os.Exit(1)
	}
	err = os.WriteFile("listings.json", listingsB, 0644)
	if err != nil {
		slog.Error(fmt.Sprintf("error in writing listings to JSON file: %v", err))
		os.Exit(1)
	}

}
