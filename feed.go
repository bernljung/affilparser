package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
)

type feed struct {
	ID                 int64
	Name               string
	URL                string
	ProductsField      string
	NameField          string
	IdentifierField    string
	DescriptionField   string
	PriceField         string
	ProductURLField    string
	RegularPriceField  string
	CurrencyField      string
	ShippingPriceField string
	InStockField       string
	GraphicURLField    string
	CategoriesField    string
	FeedData           []byte
	Categories         []category
	AllCategories      []category
	Products           map[string]product
	EntitiesCount      int
	DBOperationDone    chan string
	DBOperationError   chan error
}

type Map map[string]interface{}

func (f feed) String() (s string) {
	b, err := json.Marshal(f)
	if err != nil {
		s = ""
		return
	}
	s = string(b)
	return
}

func (f feed) update(s *session) {
	err := f.fetch()
	if err != nil {
		s.FeedError <- err
		return
	}

	err = f.parse()
	if err != nil {
		s.FeedError <- err
		return
	}

	f.DBOperationDone = make(chan string, len(f.Categories)+len(f.Products))
	f.DBOperationError = make(chan error, len(f.Categories)+len(f.Products))

	err = f.syncDB(s)
	if err != nil {
		s.FeedError <- err
		return
	}

	log.Println("Synced " + strconv.Itoa(f.EntitiesCount) + " entities")

	for i := 0; i < f.EntitiesCount; i++ {
		select {
		case result := <-f.DBOperationDone:
			log.Println(result)
		case err := <-f.DBOperationError:
			log.Println(err)
		}
	}

	s.FeedDone <- f
}

// fetch downloads the feed data
func (f *feed) fetch() error {
	resp, err := http.Get(f.URL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	f.FeedData, _ = ioutil.ReadAll(resp.Body)
	return nil
}

// parse extracts product structs and categories strings from a feed
func (f *feed) parse() error {
	var err error
	var jsonData map[string]interface{}

	// Try json first...
	err = json.Unmarshal(f.FeedData, &jsonData)
	if err == nil {
		products, ok := jsonData[f.ProductsField].([]interface{})
		if ok {
			f.Products = make(map[string]product)
			for i, _ := range products {
				p := product{}
				p.parse(products[i], f)
				f.Products[p.Identifier] = p
			}
		}
		return nil
	}

	return nil
}

func (f *feed) syncDB(s *session) error {
	var err error
	err = f.syncCategories(s)
	if err != nil {
		return err
	}

	err = f.syncProducts(s)
	if err != nil {
		return err
	}
	return nil
}

// syncCategories selects the categories from database and inserts new ones.
func (f *feed) syncCategories(s *session) error {
	dbCategories, err := s.selectCategories()
	if err != nil {
		return err
	}

	newCategories := []category{}
	for _, c := range f.Categories {
		dbIndexes := c.indexesOf(dbCategories)
		if len(dbIndexes) == 0 {
			newCategories = append(newCategories, c)
		}
	}

	for _, c := range newCategories {
		c.DBAction = DBACTION_INSERT
		m := message{feed: f, entity: c}
		f.EntitiesCount++
		s.DBOperation <- m
	}

	return nil
}

// syncProducts syncs the feed products with the products from database.
func (f *feed) syncProducts(s *session) error {
	var err error
	dbProducts, err := s.selectProducts(f.ID)
	if err != nil {
		return err
	} else {
		// Check if product exists in DB, update or insert appropriately
		for k, p := range f.Products {
			_, ok := dbProducts[k]
			if ok {
				p.ID = dbProducts[k].ID
				identicalCategories, _ := dbProducts[k].identicalCategories(
					s, p.Categories)
				if !identicalCategories {
					p.DBAction = DBACTION_UPDATE
				}

				if dbProducts[k].Name != p.Name {
					p.DBAction = DBACTION_UPDATE
				}

				if dbProducts[k].Identifier != p.Identifier {
					p.DBAction = DBACTION_UPDATE
				}

				if dbProducts[k].Description != p.Description {
					p.DBAction = DBACTION_UPDATE
				}

				if dbProducts[k].Price != dbProducts[k].Price {
					p.DBAction = DBACTION_UPDATE
				}

				if dbProducts[k].RegularPrice != dbProducts[k].RegularPrice {
					p.DBAction = DBACTION_UPDATE
				}

				if dbProducts[k].Currency != dbProducts[k].Currency {
					p.DBAction = DBACTION_UPDATE
				}

				if dbProducts[k].ShippingPrice != dbProducts[k].ShippingPrice {
					p.DBAction = DBACTION_UPDATE
				}

				if dbProducts[k].InStock != dbProducts[k].InStock {
					p.DBAction = DBACTION_UPDATE
				}

				if dbProducts[k].ProductURL != p.ProductURL {
					p.DBAction = DBACTION_UPDATE
				}

				if dbProducts[k].GraphicURL != p.GraphicURL {
					p.DBAction = DBACTION_UPDATE
				}
			} else {
				p.DBAction = DBACTION_INSERT
			}

			if p.DBAction > 0 {
				p.FeedID = f.ID
				m := message{feed: f, entity: p}
				f.EntitiesCount++
				s.DBOperation <- m
			}
		}

		// Check if DBProduct no longer exists in feed, delete
		for k, p := range dbProducts {
			_, ok := f.Products[k]
			if !ok {
				p.DBAction = DBACTION_DELETE
				m := message{feed: f, entity: p}
				f.EntitiesCount++
				s.DBOperation <- m
			}
		}
	}

	return nil
}
