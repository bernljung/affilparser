package main

import (
	"database/sql"
	"log"
	"sync"
	"time"
)

const DBACTION_INSERT = 1
const DBACTION_UPDATE = 2
const DBACTION_DELETE = 3
const DBACTION_NOTHING = 4

type session struct {
	db                        *sql.DB
	selectFeedStmt            *sql.Stmt
	insertProductStmt         *sql.Stmt
	selectCategoryStmt        *sql.Stmt
	insertCategoryStmt        *sql.Stmt
	selectProductStmt         *sql.Stmt
	deleteProductStmt         *sql.Stmt
	selectCategoryProductStmt *sql.Stmt
	insertCategoryProductStmt *sql.Stmt
	deleteCategoryProductStmt *sql.Stmt
	feeds                     []feed
	categories                []category
	DBOperation               chan message
	FeedDone                  chan feed
	FeedError                 chan error
}

func (s *session) init(dbString string) error {
	var err error
	// This does not really open a new connection.
	s.db, err = sql.Open("mysql",
		DSN+dbString)
	if err != nil {
		log.Println("Error on initializing database connection: %s",
			err.Error())
	}

	s.db.SetMaxOpenConns(50)

	// This DOES open a connection if necessary.
	// This makes sure the database is accessible.
	err = s.db.Ping()
	if err != nil {
		log.Println("Error on opening database connection: %s",
			err.Error())
	} else {
		s.prepareSelectFeedsStmt()
		s.prepareSelectCategoryStmt()
		s.prepareSelectProductStmt()
		s.prepareSelectCategoryProductStmt()
	}

	return err
}

func (s *session) prepareSelectFeedsStmt() {
	var err error
	s.selectFeedStmt, err = s.db.Prepare(
		"SELECT id, name, url, products_field, name_field, identifier_field," +
			"description_field, price_field, producturl_field, " +
			"regular_price_field, currency_field, shipping_price_field, " +
			"in_stock_field, graphicurl_field, categories_field FROM feeds")
	if err != nil {
		log.Println(err)
	}
}

func (s *session) prepareSelectCategoryStmt() {
	var err error
	s.selectCategoryStmt, err = s.db.Prepare("SELECT id, name FROM categories")
	if err != nil {
		log.Println(err)
	}
}

func (s *session) prepareSelectProductStmt() {
	var err error
	s.selectProductStmt, err = s.db.Prepare(
		"SELECT id, feed_id, name, identifier, price,  regular_price," +
			" description, currency, url, graphic_url, shipping_price, " +
			"in_stock FROM products WHERE feed_id = ?")
	if err != nil {
		log.Println(err)
	}
}

func (s *session) prepareSelectCategoryProductStmt() {
	var err error
	s.selectCategoryProductStmt, err = s.db.Prepare(
		"SELECT cp.category_id, c.name, cp.id FROM categories c " +
			"INNER JOIN category_product AS cp " +
			"ON c.`id` = cp.`category_id` " +
			"WHERE cp.`product_id` = ?")
	if err != nil {
		log.Println(err)
	}
}

func (s *session) getFeeds() error {
	s.feeds = []feed{}
	rows, err := s.selectFeedStmt.Query()
	if err != nil {
		log.Fatal(err)
	}

	defer rows.Close()
	for rows.Next() {
		f := feed{}
		err = rows.Scan(
			&f.ID,
			&f.Name,
			&f.URL,
			&f.ProductsField,
			&f.NameField,
			&f.IdentifierField,
			&f.DescriptionField,
			&f.PriceField,
			&f.ProductURLField,
			&f.RegularPriceField,
			&f.CurrencyField,
			&f.ShippingPriceField,
			&f.InStockField,
			&f.GraphicURLField,
			&f.CategoriesField,
		)

		if err != nil {
			log.Println(err)
		} else {
			s.feeds = append(s.feeds, f)
		}
	}

	if err := rows.Err(); err != nil {
		log.Fatal(err)
	}
	return err
}

func (s *session) run() {
	defer s.db.Close()
	s.FeedDone = make(chan feed, len(s.feeds))
	s.FeedError = make(chan error, len(s.feeds))
	s.DBOperation = make(chan message)

	// Run 5 worker instances for db actions
	for i := 0; i < 5; i++ {
		go s.worker()
	}

	for _, f := range s.feeds {
		go f.update(s)
	}

	for i := 0; i < len(s.feeds); i++ {
		select {
		case feed := <-s.FeedDone:
			log.Println(feed.Name + " done.")
		case err := <-s.FeedError:
			log.Println(err)
		}
	}
}

func (s *session) worker() {
	var wg sync.WaitGroup
	for {
		select {
		case message := <-s.DBOperation:
			switch message.entity.getDBAction() {

			case DBACTION_INSERT:
				wg.Add(1)
				go func() {
					defer wg.Done()
					message.entity.insert(message.feed, s)
				}()

			case DBACTION_UPDATE:
				wg.Add(1)
				go func() {
					defer wg.Done()
					message.entity.update(message.feed, s)
				}()

			case DBACTION_DELETE:
				wg.Add(1)
				go func() {
					defer wg.Done()
					message.entity.delete(message.feed, s)
				}()

			default:
				time.Sleep(1 * time.Millisecond)
			}
		default:
			time.Sleep(1 * time.Millisecond)
		}
		wg.Wait()
	}
}

func (s *session) selectCategories() ([]category, error) {
	categories := []category{}
	rows, err := s.selectCategoryStmt.Query()
	if err != nil {
		return categories, err
	}

	defer rows.Close()
	for rows.Next() {
		c := category{}
		if err := rows.Scan(&c.ID, &c.Name); err == nil {
			categories = append(categories, c)
		} else {
			log.Fatal(err)
		}
	}
	if err := rows.Err(); err != nil {
		log.Fatal(err)
	}
	return categories, err
}

func (s *session) selectProducts(feedID int64) (map[string]product, error) {
	products := make(map[string]product)

	rows, err := s.selectProductStmt.Query(feedID)
	if err != nil {
		return products, err
	}

	defer rows.Close()
	for rows.Next() {
		p := product{}
		err := rows.Scan(
			&p.ID,
			&p.FeedID,
			&p.Name,
			&p.Identifier,
			&p.Price,
			&p.RegularPrice,
			&p.Description,
			&p.Currency,
			&p.ProductURL,
			&p.GraphicURL,
			&p.ShippingPrice,
			&p.InStock,
		)

		if err != nil {
			return products, err
		} else {
			products[p.Identifier] = p
		}
	}

	err = rows.Err()

	return products, err
}
