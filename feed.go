package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
)

type feedmessage struct {
	feed   *feed
	err    error
	action string
}

type feed struct {
	ID                    int64
	Name                  string
	URL                   string
	ProductsField         string
	NameField             string
	IdentifierField       string
	DescriptionField      string
	PriceField            string
	ProductURLField       string
	RegularPriceField     string
	CurrencyField         string
	ShippingPriceField    string
	InStockField          string
	GraphicURLField       string
	CategoriesField       string
	SyncCategories        bool
	AllowEmptyDescription bool
	FeedData              []byte
	Categories            []categoryinterface
	AllCategories         []categoryinterface
	Products              map[string]product
	EntitiesCount         int
	DBOperationDone       chan string
	DBOperationError      chan error
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

func (f *feed) update(s *session) {
	err := f.fetch()
	if err != nil {
		log.Println(err)
		s.FeedError <- feedmessage{feed: f, err: err, action: "update"}
		return
	}

	err = f.parse()
	if err != nil {
		log.Println(err)
		s.FeedError <- feedmessage{feed: f, err: err, action: "update"}
		return
	}

	f.DBOperationDone = make(chan string, len(f.Categories)+len(f.Products))
	f.DBOperationError = make(chan error, len(f.Categories)+len(f.Products))

	err = f.syncDB(s)
	if err != nil {
		log.Println(err)
		s.FeedError <- feedmessage{feed: f, err: err, action: "update"}
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

	s.FeedDone <- feedmessage{feed: f, err: nil, action: "update"}
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

	err = json.Unmarshal(f.FeedData, &jsonData)
	if err != nil {
		log.Println(err)
		return err
	}
	products, ok := jsonData[f.ProductsField].([]interface{})
	if ok {
		f.Products = make(map[string]product)
		for i, _ := range products {
			p := product{}
			p.parse(products[i], f)
			valid := p.validate(f)
			if valid {
				f.Products[p.Identifier] = p
			}
		}
	}
	return nil
}

func (f *feed) refresh(s *session) {
	var err error
	f.Products, err = f.selectProducts(s)
	f.DBOperationDone = make(chan string, len(f.Products))
	f.DBOperationError = make(chan error, len(f.Products))

	if err != nil {
		log.Println(err)
	}

	for _, p := range f.Products {
		refreshed := p.refresh()
		if refreshed == true {
			p.FeedID = f.ID
			m := message{feed: f, entity: p}
			f.EntitiesCount++
			s.DBOperation <- m
		}
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

	s.FeedDone <- feedmessage{feed: f, err: nil, action: "updateProductsKeywords"}
}

func (f *feed) syncProductCategories(s *session, update bool) {
	products, err := f.selectProducts(s)
	if err != nil {
		log.Println(err)
	}

	for _, p := range products {
		_ = p.syncCategories(s, f, update)
	}
	s.FeedDone <- feedmessage{feed: f, err: nil, action: "syncCategories"}
}

func (f *feed) syncDB(s *session) error {
	var err error

	if f.SyncCategories {
		err = f.syncCategories(s)
	} else {
		err = f.deleteFeedCategories(s)
	}

	if err != nil {
		log.Println(err)
		return err
	}

	err = f.syncProducts(s)
	if err != nil {
		log.Println(err)
		return err
	}
	return nil
}

func (f *feed) deleteFeedCategories(s *session) error {
	dbCategories, err := s.selectCategories()

	if err != nil {
		log.Println(err)
		return err
	}

	categoriesToDelete := []categoryinterface{}
	for _, c := range f.Categories {
		indexes := c.indexesOf(dbCategories)
		if len(indexes) > 0 {
			for _, i := range indexes {
				category := dbCategories[i]
				createdByID := category.getCreatedByID()
				descriptionByUser := category.getDescriptionByUser()
				keywords := category.getKeywords()
				if createdByID == CREATED_BY_FEED && descriptionByUser == "" && keywords == "" {
					categoriesToDelete = append(categoriesToDelete, category)
				}
			}
		}
	}

	for _, c := range categoriesToDelete {
		c.setDBAction(DBACTION_DELETE)
		m := message{feed: f, entity: c}
		f.EntitiesCount++
		s.DBOperation <- m
	}

	return nil
}

// syncCategories selects the categories from database and inserts new ones.
func (f *feed) syncCategories(s *session) error {
	dbCategories, err := s.selectCategories()
	if err != nil {
		log.Println(err)
		return err
	}

	newCategories := []categoryinterface{}
	for _, c := range f.Categories {
		indexes := c.indexesOf(dbCategories)
		if len(indexes) == 0 {
			newCategories = append(newCategories, c)
		}
	}

	for _, c := range newCategories {
		c.setDBAction(DBACTION_INSERT)
		m := message{feed: f, entity: c}
		f.EntitiesCount++
		s.DBOperation <- m
	}

	return nil
}

func (f feed) selectProducts(s *session) (map[string]product, error) {
	products := make(map[string]product)

	rows, err := s.selectProductStmt.Query(f.ID)
	if err != nil {
		log.Println(err)
		return products, err
	}

	defer rows.Close()
	for rows.Next() {
		p := product{}
		err := rows.Scan(
			&p.ID,
			&p.FeedID,
			&p.Name,
			&p.NameByUser,
			&p.Identifier,
			&p.Price,
			&p.RegularPrice,
			&p.Description,
			&p.DescriptionByUser,
			&p.Keywords,
			&p.Currency,
			&p.ProductURL,
			&p.GraphicURL,
			&p.ShippingPrice,
			&p.InStock,
		)

		if err != nil {
			log.Println(err)
			return products, err
		} else {
			products[p.Identifier] = p
		}
	}

	err = rows.Err()

	return products, err
}

// syncProducts syncs the feed products with the products from database.
func (f *feed) syncProducts(s *session) error {
	var err error
	dbProducts, err := f.selectProducts(s)
	if err != nil {
		log.Println(err)
		return err
	} else {
		// Check if product exists in DB, update or insert appropriately
		for k, p := range f.Products {
			_, ok := dbProducts[k]

			_ = p.refresh()

			if ok {
				p.ID = dbProducts[k].ID

				if dbProducts[k].Name != p.Name {
					log.Println("ShippinNamegPrice")
					p.DBAction = DBACTION_UPDATE
				}

				if dbProducts[k].Identifier != p.Identifier {
					log.Println("Identifier")
					p.DBAction = DBACTION_UPDATE
				}

				if dbProducts[k].Description != p.Description {
					log.Println("Description")
					p.DBAction = DBACTION_UPDATE
				}

				if dbProducts[k].Price != dbProducts[k].Price {
					log.Println("Price")
					p.DBAction = DBACTION_UPDATE
				}

				if dbProducts[k].RegularPrice != dbProducts[k].RegularPrice {
					log.Println("RegularPrice")
					p.DBAction = DBACTION_UPDATE
				}

				if dbProducts[k].Currency != dbProducts[k].Currency {
					log.Println("Currency")
					p.DBAction = DBACTION_UPDATE
				}

				if dbProducts[k].ShippingPrice != dbProducts[k].ShippingPrice {
					log.Println("ShippingPrice")
					p.DBAction = DBACTION_UPDATE
				}

				if dbProducts[k].InStock != dbProducts[k].InStock {
					log.Println("InStock")
					p.DBAction = DBACTION_UPDATE
				}

				if dbProducts[k].ProductURL != p.ProductURL {
					log.Println("ProductURL")
					p.DBAction = DBACTION_UPDATE
				}

				if dbProducts[k].GraphicURL != p.GraphicURL {
					log.Println("GraphicURL")
					p.DBAction = DBACTION_UPDATE
				}

				if dbProducts[k].Keywords != p.Keywords {
					log.Println("Keywords")
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
