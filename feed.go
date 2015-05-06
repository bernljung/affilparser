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
	ID                    int
	SiteID                int
	Name                  string
	URL                   string
	NetworkID             int
	Network               networkinterface
	AllowEmptyDescription bool
	FeedData              []byte
	Products              map[string]product
	ProductsCount         int
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

	f.DBOperationDone = make(chan string, len(f.Products))
	f.DBOperationError = make(chan error, len(f.Products))

	err = f.syncProducts(s)
	if err != nil {
		log.Println(err)
		s.FeedError <- feedmessage{feed: f, err: err, action: "update"}
		return
	}

	log.Println("Synced " + strconv.Itoa(f.ProductsCount) + " products")

	for i := 0; i < f.ProductsCount; i++ {
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

func (f *feed) parse() error {
	var err error

	products, err := f.Network.parseProducts(f)
	if err == nil {
		f.Products = make(map[string]product)
		for i, _ := range products {
			f.Products[products[i].Identifier] = products[i]
		}
	}
	return err
}

func (f feed) selectNetwork(s *session) (networkinterface, error) {
	var err error
	var network networkinterface

	switch f.NetworkID {
	case NETWORK_ADRECORD:
		n := adrecord{}
		network = n

	case NETWORK_TRADEDOUBLER:
		n := tradedoubler{}
		network = n
	default:
		log.Println("Invalid network id")
		return network, err
	}

	return network, err
}

func (f feed) selectProducts(s *session) (map[string]product, error) {
	products := make(map[string]product)

	rows, err := s.selectFeedProductStmt.Query(f.ID)
	if err != nil {
		log.Println(err)
		return products, err
	}

	defer rows.Close()
	for rows.Next() {
		p := product{}
		err := rows.Scan(
			&p.ID,
			&p.SiteID,
			&p.FeedID,
			&p.Name,
			&p.NameByUser,
			&p.Identifier,
			&p.Price,
			&p.RegularPrice,
			&p.Description,
			&p.DescriptionByUser,
			&p.Currency,
			&p.ProductURL,
			&p.GraphicURL,
			&p.ShippingPrice,
			&p.InStock,
			&p.Points,
			&p.HasCategories,
			&p.Active,
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

			if ok {
				p.ID = dbProducts[k].ID

				if p.Description == "" && f.AllowEmptyDescription == false {
					p.DBAction = DBACTION_DELETE

				} else {
					if dbProducts[k].Name != p.Name {
						log.Println("Name")
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
				}

			} else {
				p.DBAction = DBACTION_INSERT
			}

			if p.DBAction > 0 {
				p.FeedID = f.ID
				p.SiteID = f.SiteID
				m := message{feed: f, product: p}
				f.ProductsCount++
				s.DBOperation <- m
			} else {
				log.Println("No action for " + p.Name)
			}
		}

		// Check if DBProduct no longer exists in feed, delete
		for k, p := range dbProducts {
			_, ok := f.Products[k]
			if !ok {
				p.DBAction = DBACTION_DELETE

				p.SiteID = f.SiteID
				m := message{feed: f, product: p}
				f.ProductsCount++
				s.DBOperation <- m
			}
		}
	}

	return nil
}
