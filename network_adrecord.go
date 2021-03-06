package main

import (
	"encoding/json"
	"log"
	"strconv"
	"strings"
)

type adrecord struct {
	Total    int
	Products []struct {
		Name          string
		SKU           string
		EAN           string
		Description   string
		Model         string
		Brand         string
		Gender        string
		Price         string
		RegularPrice  string
		ShippingPrice string
		Currency      string
		ProductURL    string
		GraphicURL    string
		InStock       string
		InStockQty    string
		DeliveryTime  string
	}
}

func (n adrecord) parseProducts(f *feed) ([]product, error) {
	var err error
	var products []product

	// Decode the json object
	a := &adrecord{}
	err = json.Unmarshal([]byte(f.FeedData), &a)
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

		p.RegularPrice, errs = strconv.ParseFloat(v.RegularPrice, 64)
		if errs != nil {
			p.RegularPrice = 0
			errs = nil
		}

		p.Description = v.Description
		p.Currency = v.Currency
		p.ProductURL = v.ProductURL
		p.GraphicURL = v.GraphicURL
		p.ShippingPrice, errs = strconv.ParseFloat(v.ShippingPrice, 64)
		if errs != nil {
			p.ShippingPrice = 0
			errs = nil
		}

		p.InStock, _ = strconv.ParseBool(v.InStock)
		p.SiteID = f.SiteID
		p.FeedID = f.ID

		products = append(products, p)
	}

	return products, err
}
