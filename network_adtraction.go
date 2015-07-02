package main

import (
	"encoding/xml"
	"log"
	"strconv"
	"strings"
)

type AdtractionProduct struct {
	SKU         string
	Name        string
	Description string
	Category    string
	Price       string
	Shipping    string
	Currency    string
	InStock     string
	ProductUrl  string
	ImageUrl    string
	TrackingUrl string
}

type adtraction struct {
	Products []AdtractionProduct `xml:"product"`
}

func (n adtraction) parseProducts(f *feed) ([]product, error) {
	var err error
	var products []product

	// Decode the json object
	a := &adtraction{}
	err = xml.Unmarshal([]byte(f.FeedData), &a)
	if err != nil {
		log.Println(err)
	}

	for _, v := range a.Products {
		var errs error
		p := product{}
		p.Name = strings.Replace(v.Name, "&quot;", "", -1)
		p.Slug = generateSlug(p.Name)
		p.Identifier = v.SKU
		p.Price, errs = strconv.ParseFloat(v.Price, 64)
		if errs != nil {
			p.Price = 0
			errs = nil
		}

		p.RegularPrice, errs = strconv.ParseFloat(v.Price, 64)
		if errs != nil {
			p.RegularPrice = 0
			errs = nil
		}

		p.Description = v.Description
		p.Currency = v.Currency
		p.ProductURL = v.TrackingUrl
		p.GraphicURL = v.ImageUrl
		p.ShippingPrice, errs = strconv.ParseFloat(v.Shipping, 64)
		if errs != nil {
			p.ShippingPrice = 0
			errs = nil
		}

		if v.InStock == "yes" {
			p.InStock = true
		}
		p.SiteID = f.SiteID
		p.FeedID = f.ID

		products = append(products, p)
	}

	return products, err
}
