package property

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	solver "github.com/idea456/commuter-scraper/internal/solver"

	"github.com/PuerkitoBio/goquery"
	_ "gocloud.dev/blob/s3blob"
)

type Coordinate struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type Property struct {
	PropertyId  string     `json:"PropertyId"`
	District    string     `json:"district"`
	Region      string     `json:"region"`
	Name        string     `json:"name"`
	Address     string     `json:"address"`
	Facilities  []string   `json:"facilities"`
	Link        string     `json:"link"`
	Type        string     `json:"type"`
	Coordinates Coordinate `json:"coordinates"`
}

type PropertyScraper struct {
	solver  *solver.Solver
	minPage int
	maxPage int
}

func NewPropertyScraper(solver *solver.Solver, minPage int, maxPage int) (*PropertyScraper, error) {
	return &PropertyScraper{
		minPage: minPage,
		maxPage: maxPage,
		solver:  solver,
	}, nil
}

func (s *PropertyScraper) ScrapePropertyLinks(rawHtml string) ([]string, error) {
	// Load the HTML document
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(rawHtml))
	if err != nil {
		return nil, err
	}

	propertyLinks := make([]string, 0)
	container := doc.Find(".main-content")
	container.Find(".header-container").Each(func(_ int, el *goquery.Selection) {
		propertyLink, exists := el.Find("h3 .nav-link").Attr("href")
		if !exists {
			return
		}

		propertyLinks = append(propertyLinks, fmt.Sprintf("https://www.propertyguru.com.my%s", propertyLink))
	})

	return propertyLinks, nil
}

func (s *PropertyScraper) ScrapeProperty(ctx context.Context, rawHtml string, propertyLink string) (*Property, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(rawHtml))
	if err != nil {
		return nil, err
	}

	container := doc.Find("#wrapper")

	breadcrumbs := container.Find(".container ol.breadcrumb li")
	var district string
	var region string
	breadcrumbs.Each(func(i int, el *goquery.Selection) {
		if i == breadcrumbs.Length()-2 {
			district = strings.TrimSpace(el.Find("a span").Text())
		} else if i == breadcrumbs.Length()-3 {
			region = strings.TrimSpace(el.Find("a span").Text())
		}
	})

	address := container.Find(".listing-address span").Text()

	var name string
	var propertyType string

	table := container.Find(".listing-details-primary table")
	table.Find("tbody").Each(func(i int, row *goquery.Selection) {
		title := row.Find("tr.property-attr td.label-block h4.label-block").Text()
		switch strings.ToLower(title) {
		case "project name":
			name = row.Find("tr.property-attr td.value-block").Text()
		case "project type":
			propertyType = row.Find("tr.property-attr td.value-block").Text()
		}
	})

	facilities := make([]string, 0)
	facilitiesTable := container.Find("#facilities ul")
	facilitiesTable.Find("li").Each(func(i int, row *goquery.Selection) {
		facility := row.Find("span").Text()
		facilities = append(facilities, facility)
	})

	mapContainer := container.Find("#map")
	var coordinates Coordinate
	mapContainer.Find("meta").Each(func(i int, el *goquery.Selection) {
		attr := el.AttrOr("itemprop", "")
		content := el.AttrOr("content", "")
		if attr != "" {
			if attr == "latitude" {
				coordinates.Latitude, err = strconv.ParseFloat(content, 64)
				if err != nil {
					slog.Error(fmt.Sprintf("could not parse float %v for property %s: %v", content, name, err))
				}
			} else if attr == "longitude" {
				coordinates.Longitude, err = strconv.ParseFloat(content, 64)
				if err != nil {
					slog.Error(fmt.Sprintf("could not parse float %v for property %s: %v", content, name, err))
				}
			}
		}
	})

	return &Property{
		Name:        name,
		Type:        propertyType,
		Address:     address,
		Coordinates: coordinates,
		Facilities:  facilities,
		Link:        propertyLink,
		District:    district,
		Region:      region,
	}, nil
}

func (s *PropertyScraper) Scrape(ctx context.Context, propertyLinks []string) ([]Property, error) {
	properties := make([]Property, 0)
	for i := 0; i < len(propertyLinks); i++ {
		rawHtml, err := s.solver.RequestPage(propertyLinks[i])
		if err != nil {
			slog.Error(fmt.Sprintf("could not fetch property %s: %v", propertyLinks[i], err))
			return properties, err
		}

		property, err := s.ScrapeProperty(ctx, rawHtml, propertyLinks[i])
		if err != nil {
			slog.Error(fmt.Sprintf("could not parse property HTML %s: %v", propertyLinks[i], err))
			continue
		}

		properties = append(properties, *property)
	}

	slog.Info(fmt.Sprintf("Total properties scraped: %d", len(properties)))

	return properties, nil
}
