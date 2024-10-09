package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"log/slog"
	"os"
	"strconv"
	"strings"

	"github.com/idea456/commuter-scraper/internal/property"
	"github.com/idea456/commuter-scraper/internal/solver"
	_ "gocloud.dev/blob/s3blob"
)

func main() {
	if len(os.Args) < 3 {
		log.Fatal(fmt.Errorf("please specify min and max pages!"))
		return
	}

	minPages, err := strconv.Atoi(os.Args[1])
	if err != nil {
		log.Fatal(fmt.Errorf("min pages should be a number!"))
		return
	}

	maxPages, err := strconv.Atoi(os.Args[2])
	if err != nil {
		log.Fatal(fmt.Errorf("max pages should be a number!"))
		return
	}

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

	scraper, err := property.NewPropertyScraper(solver, minPages, maxPages)
	if err != nil {
		slog.Error(fmt.Sprintf("unable to create property scraper: %v", err))
		return
	}

	properties, err := scraper.ScrapePropertyLinks(ctx)
	if err != nil {
		slog.Error(fmt.Sprintf("unable to scrape properties: %v", err))
	}

	propertiesStr := strings.Join(properties, "\n")

	err = ioutil.WriteFile("property-links.txt", []byte(propertiesStr), 0777)
	// handle this error
	if err != nil {
		// print it out
		fmt.Println(err)
	}
}
