package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/idea456/commuter-scraper/internal/property"
	"github.com/idea456/commuter-scraper/internal/solver"
	"gocloud.dev/blob"
	_ "gocloud.dev/blob/s3blob"
)

func main() {
	bucketUrl := os.Getenv("S3_BUCKET_URL")
	if bucketUrl == "" {
		slog.Error("S3_BUCKET_URL is missing!")
		return
	}

	solverUrl := os.Getenv("SOLVER_URL")
	if solverUrl == "" {
		slog.Error("SOLVER_URL is missing!")
		return
	}

	slog.SetLogLoggerLevel(slog.LevelDebug)
	ctx := context.Background()

	solver, err := solver.NewSolver(solverUrl)
	if err != nil {
		slog.Error(fmt.Sprintf("unable to initialise solver: %v", err))
		return
	}
	defer solver.DestroySession()

	scraper, err := property.NewPropertyScraper(solver, 0, 0)
	if err != nil {
		slog.Error(fmt.Sprintf("unable to create listing scraper: %v", err))
		return
	}

	file, err := os.ReadFile("property-links.txt")
	if err != nil {
		slog.Error(fmt.Sprintf("could not open properties links file: %v", err))
		return
	}

	propertyLinks := strings.Split(string(file), "\n")

	properties, err := scraper.Scrape(ctx, propertyLinks)
	if err != nil {
		slog.Error(fmt.Sprintf("unable to scrape properties: %v", err))
		return
	}

	bucket, err := blob.OpenBucket(ctx, bucketUrl)
	if err != nil {
		log.Fatal(fmt.Errorf("could not open bucket: %v", err))
		return
	}
	defer bucket.Close()

	timestamp := strconv.FormatInt(time.Now().UTC().UnixNano(), 10)
	scrapedPropertiesFileName := fmt.Sprintf("properties-%s.json", timestamp)
	w, err := bucket.NewWriter(ctx, scrapedPropertiesFileName, nil)
	if err != nil {
		log.Fatal(fmt.Errorf("could not open bucket: %v", err))
		// return
	}

	writeErr := json.NewEncoder(w).Encode(properties)
	// Always check the return value of Close when writing.
	closeErr := w.Close()
	if writeErr != nil {
		log.Fatal(writeErr)
	}
	if closeErr != nil {
		log.Fatal(closeErr)
	}

	propertiesB, err := json.Marshal(properties)
	if err != nil {
		slog.Error(fmt.Sprintf("error in marshalling listings to bytes: %v", err))
		os.Exit(1)
	}

	err = os.WriteFile(scrapedPropertiesFileName, propertiesB, 0644)
	if err != nil {
		slog.Error(fmt.Sprintf("error in writing properties to JSON file: %v", err))
		os.Exit(1)
	}
}
