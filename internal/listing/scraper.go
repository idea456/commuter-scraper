package listing

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

type Listing struct {
	Name         string    `json:"name"`
	Address      string    `json:"address"`
	Link         string    `json:"link"`
	PropertyId   string    `json:"propertyId"`
	Price        int       `json:"price"`
	Currency     string    `json:"currency"`
	Period       string    `json:"period"`
	PSF          float64   `json:"psf"`
	Area         string    `json:"area"`
	Furnished    string    `json:"furnished"`
	Amenities    Amenities `json:"amenities"`
	PropertyType string    `json:"propertyType"`
}

type Amenities struct {
	Studio    bool `json:"studio"`
	Bathrooms int  `json:"bathrooms"`
	Bedrooms  int  `json:"bedrooms"`
}

type ListingScraper struct {
	solver  *solver.Solver
	minPage int
	maxPage int
}

func NewListingScraper(solver *solver.Solver, minPage int, maxPage int) (*ListingScraper, error) {
	return &ListingScraper{
		minPage: minPage,
		maxPage: maxPage,
		solver:  solver,
	}, nil
}

func (s *ListingScraper) scrapeListing(rawHtml string) (*Listing, error) {
	// Load the HTML document
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(rawHtml))
	if err != nil {
		return nil, err
	}

	var listing Listing
	container := doc.Find("#listings-container")
	container.Find(".listing-card").Each(func(_ int, el *goquery.Selection) {
		title := el.Find(".header-wrapper .header-container a.nav-link").AttrOr("title", "")
		link := el.Find(".header-wrapper .header-container a.nav-link").AttrOr("href", "")
		address := el.Find("p.listing-location span").Text()
		currency := el.Find(".listing-features .list-price .currency").Text()
		price := el.Find(".listing-features .list-price .price").Text()
		period := el.Find(".listing-features .list-price .period").Text()

		var furnished string
		var propertyType string
		propertyTypes := el.Find(".listing-properties .listing-property-type li")
		propertyTypes.Each(func(_ int, propertyTypeEl *goquery.Selection) {
			text := propertyTypeEl.Text()
			if strings.Contains(text, "Furnished") {
				furnished = text
			} else if !strings.Contains(text, "Completion") {
				propertyType = text
			}
		})

		var psf float64
		var area string
		areaEl := el.Find(".listing-features .listing-floorarea")

		areaEl.Each(func(i int, elArea *goquery.Selection) {
			if i == 0 {
				psfStr := strings.Split(elArea.Text(), " ")[0]
				psfInt, err := strconv.Atoi(psfStr)
				if err != nil {
					slog.Error(fmt.Sprintf("unable to cast PSF %s to string", psfStr))
				} else {
					psf = float64(psfInt)
				}
			} else {
				area = elArea.Text()

			}
		})

		var studio bool = false
		bathroomsStr := el.Find(".listing-features li.listing-rooms span.bath").AttrOr("title", "0")
		bedroomsStr := el.Find(".listing-features li.listing-rooms span.bed").AttrOr("title", "0")

		bathrooms, err := strconv.Atoi(strings.Split(bathroomsStr, " ")[0])
		if err != nil {
			slog.Error(fmt.Sprintf("unable to parse bathrooms text '%s' to int: %v", bathroomsStr, err))
		}
		bedrooms, err := strconv.Atoi(strings.Split(bedroomsStr, " ")[0])
		if err != nil {
			slog.Error(fmt.Sprintf("unable to parse bedrooms text '%s' for %s to int: %v", bedroomsStr, link, err))
		}

		priceInt, err := strconv.Atoi(strings.ReplaceAll(strings.Trim(price, " "), ",", ""))
		if err != nil {
			slog.Error(fmt.Sprintf("unable to parse price %s for listing %s", price, link))
		}

		listing = Listing{
			Name:         title,
			Link:         link,
			Address:      address,
			Currency:     strings.Trim(currency, " "),
			Price:        priceInt,
			Period:       period,
			Furnished:    furnished,
			PropertyType: propertyType,
			PSF:          psf,
			Area:         area,
			Amenities: Amenities{
				Bedrooms:  bedrooms,
				Bathrooms: bathrooms,
				Studio:    studio,
			},
		}

	})

	return &listing, nil
}

func (s *ListingScraper) Scrape(ctx context.Context) ([]Listing, error) {
	slog.Info(fmt.Sprintf("Scraping from page %d to page %d...\n", s.minPage, s.maxPage))

	listings := make([]Listing, 0)

	for page := s.minPage; page < s.maxPage; page++ {
		var listingUrl string
		if page == 1 {
			listingUrl = "https://www.propertyguru.com.my/apartment-condo-service-residence-for-rent/in-kuala-lumpur-58jok"
		} else {
			listingUrl = fmt.Sprintf("https://www.propertyguru.com.my/apartment-condo-service-residence-for-rent/in-kuala-lumpur-58jok/%d", page)
		}

		rawHtml, err := s.solver.RequestPage(listingUrl)
		if err != nil {
			return listings, err
		}

		listing, err := s.scrapeListing(rawHtml)
		if err != nil {
			return listings, err
		}
		listings = append(listings, *listing)

	}

	slog.Info(fmt.Sprintf("Total listings scraped: %d", len(listings)))

	return listings, nil
}
