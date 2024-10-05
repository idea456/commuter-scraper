package scraper

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/idea456/commuter-scraper/internal/solver"

	"github.com/gocolly/colly/v2"
	"github.com/gocolly/colly/v2/queue"
)

type ListingScraper struct {
	solverServer *solver.SolverServer
}

func NewListingScraper(solver *solver.SolverServer) *ListingScraper {
	return &ListingScraper{
		solverServer: solver,
	}
}

func (s *ListingScraper) ScrapeListing() ([]Listing, error) {
	c := colly.NewCollector(colly.AllowURLRevisit())

	listings := make([]Listing, 0)

	c.OnHTML("#listings-container", func(e *colly.HTMLElement) {
		e.ForEach(".listing-card", func(_ int, el *colly.HTMLElement) {
			title := el.ChildAttr(".header-wrapper .header-container a.nav-link", "title")
			link := e.ChildAttr(".header-wrapper .header-container a.nav-link", "href")
			address := el.ChildText("p.listing-location span")
			currency := el.ChildText(".listing-features .list-price .currency")
			price := el.ChildText(".listing-features .list-price .price")
			period := el.ChildText(".listing-features .list-price .period")

			var psf float64
			var area float64
			el.ForEach(".listing-features .listing-floorarea", func(i int, elFloorArea *colly.HTMLElement) {
				if i == 0 {
					psfStr := strings.Split(elFloorArea.Text, " ")[0]
					psfInt, err := strconv.Atoi(psfStr)
					if err != nil {
						slog.Error(fmt.Sprintf("unable to cast PSF %s to string", psfStr))
					} else {
						psf = float64(psfInt)
					}
				} else {
					areaTokens := strings.Split(elFloorArea.Text, " ")
					if len(areaTokens) >= 2 {
						areaInt, err := strconv.Atoi(areaTokens[1])
						if err != nil {
							slog.Error("unable to cast PSF %s to string", areaTokens[1])
						} else {
							psf = float64(areaInt)
						}
					}

				}
			})

			var studio bool = false
			bathroomsStr := e.ChildAttr(".listing-features li.listing-rooms span.bath", "title")
			bedroomsStr := e.ChildAttr(".listing-features li.listing-rooms span.bed", "title")

			bathrooms, err := strconv.Atoi(strings.Split(bathroomsStr, " ")[0])
			if err != nil {
				slog.Error(fmt.Sprintf("unable to parse bathrooms text '%s' to int: %v", bathroomsStr, err))
			}
			bedrooms, err := strconv.Atoi(strings.Split(bedroomsStr, " ")[0])
			if err != nil {
				slog.Error(fmt.Sprintf("unable to parse bedrooms text '%s' to int: %v", bedroomsStr, err))
			}

			priceInt, err := strconv.Atoi(strings.ReplaceAll(price, ",", ""))
			if err != nil {
				slog.Error(fmt.Sprintf("unable to parse price %s for listing %s", price, link))
			}

			listing := Listing{
				Name:     title,
				Link:     link,
				Address:  address,
				Currency: currency,
				Price:    priceInt,
				PSF:      psf,
				Area:     area,
				Period:   period,
				Amenities: Amenities{
					Bedrooms:  bedrooms,
					Bathrooms: bathrooms,
					Studio:    studio,
				},
			}

			listings = append(listings, listing)
		})
	})

	c.SetRequestTimeout(180 * time.Second)
	// DO NOT REMOVE THIS, otherwise OnHTML will not work
	c.OnResponse(func(r *colly.Response) {
		fmt.Println("scraping...")
		var body solver.ResponsePage
		json.Unmarshal(r.Body, &body)
		r.Headers.Set("Content-Type", "text/html; charset=UTF-8")

		r.Body = []byte(body.Solution.Response)
	})

	// create a request queue with 2 consumer threads
	q, _ := queue.New(
		2, // Number of consumer threads
		&queue.InMemoryQueueStorage{MaxSize: 10000}, // Use default queue storage
	)

	for page := 1; page < 10; page++ {
		var listingUrl string
		if page == 1 {
			listingUrl = "https://www.propertyguru.com.my/apartment-condo-service-residence-for-rent/in-kuala-lumpur-58jok"
		} else {
			listingUrl = fmt.Sprintf("https://www.propertyguru.com.my/apartment-condo-service-residence-for-rent/in-kuala-lumpur-58jok/%d", page)
		}
		body := solver.RequestPage{
			Cmd:        "request.get",
			Url:        listingUrl,
			MaxTimeout: 60000,
		}
		b, err := json.Marshal(body)
		if err != nil {
			slog.Error("could not parse JSON for request to ")
			return nil, err
		}

		solverUrl := url.URL{
			Scheme: "http",
			Host:   "localhost:8191",
			Path:   "/v1",
		}

		r := colly.Request{
			URL:    &solverUrl,
			Method: "POST",
			Body:   bytes.NewBuffer(b),
			Headers: &http.Header{
				"Content-Type": []string{"application/json"},
			},
		}

		q.AddRequest(&r)
	}

	q.Run(c)

	return listings, nil
}
