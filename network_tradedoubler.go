package main

import (
	"encoding/json"
	"log"
	"strconv"
	"strings"
)

type tradedoubler struct {
	ProductHeader struct {
		TotalHits int
	}
	Products []struct {
		Name         string
		ProductImage struct {
			URL string
		}
		Language    string
		Description string
		Brand       string
		Identifiers struct {
			EAN string
			MPN string
			SKU string
		}
		GroupingID string
		Fields     []struct {
			Name  string
			Value string
		}
		Offers []struct {
			FeedID       int
			ProductURL   string
			PriceHistory []struct {
				Price struct {
					Value    string
					Currency string
				}
				Date int
			}
			Modified        int
			InStock         int
			Availability    string
			ShippingCost    string
			SourceProductID string
			ProgramLogo     string
			ProgramName     string
			ID              string
		}
		Categories []struct {
			Name           string
			TDCategoryName string
			ID             int
		}
	}
}

func (n tradedoubler) parseProducts(f *feed) ([]product, error) {
	var err error
	var products []product

	// Decode the json object
	a := &tradedoubler{}
	err = json.Unmarshal([]byte(f.FeedData), &a)
	if err != nil {
		log.Println(err)
	}

	for _, v := range a.Products {
		var errs error
		p := product{}
		p.Name = strings.Replace(v.Name, "&quot;", "", -1)
		p.Slug = generateSlug(p.Name)
		p.Identifier = v.Identifiers.SKU
		p.Price, errs = strconv.ParseFloat(v.Offers[0].PriceHistory[0].Price.Value, 64)
		if errs != nil {
			p.Price = 0
			errs = nil
		}

		p.RegularPrice = p.Price
		p.Description = v.Description
		p.Currency = v.Offers[0].PriceHistory[0].Price.Currency
		p.ProductURL = v.Offers[0].ProductURL
		p.GraphicURL = v.ProductImage.URL
		p.ShippingPrice, errs = strconv.ParseFloat(v.Offers[0].ShippingCost, 64)
		if errs != nil {
			p.ShippingPrice = 0
			errs = nil
		}

		if v.Offers[0].InStock > 0 {
			p.InStock = true
		} else {
			p.InStock = false
		}
		p.SiteID = f.SiteID
		p.FeedID = f.ID

		products = append(products, p)
	}

	return products, err
}
