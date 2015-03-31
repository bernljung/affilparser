package main

import (
	"database/sql"
	"fmt"
	"log"
	"sync"
	"time"
)

const DBACTION_INSERT = 1
const DBACTION_UPDATE = 2
const DBACTION_DELETE = 3
const DBACTION_NOTHING = 4

const CREATED_BY_USER = 1
const CREATED_BY_KEYWORD = 2
const CREATED_BY_FEED = 3

type session struct {
	db                        *sql.DB
	selectSiteStmt            *sql.Stmt
	selectFeedStmt            *sql.Stmt
	insertProductStmt         *sql.Stmt
	selectCategoryStmt        *sql.Stmt
	insertCategoryStmt        *sql.Stmt
	selectFeedProductStmt     *sql.Stmt
	deleteProductStmt         *sql.Stmt
	selectCategoryProductStmt *sql.Stmt
	insertCategoryProductStmt *sql.Stmt
	deleteCategoryProductStmt *sql.Stmt
	site                      *site
	feeds                     []*feed
	categories                []categoryinterface
	DBOperation               chan message
	FeedDone                  chan feedmessage
	FeedError                 chan feedmessage
}

func (s *session) init(subdomain string) error {
	var err error
	// This does not really open a new connection.
	s.db, err = sql.Open("mysql",
		DSN)
	if err != nil {
		log.Println("Error on initializing database connection: %s",
			err.Error())
	}

	s.db.SetMaxOpenConns(1)

	// This DOES open a connection if necessary.
	// This makes sure the database is accessible.
	err = s.db.Ping()
	if err != nil {
		log.Println("Error on opening database connection: %s",
			err.Error())
	} else {
		s.prepareSelectSiteStmt()
		s.prepareSelectFeedsStmt()
		s.prepareSelectCategoryStmt()
		s.prepareSelectFeedProductStmt()
		s.prepareSelectCategoryProductStmt()
	}

	s.selectSite(subdomain)
	return err
}

func (s *session) prepareSelectSiteStmt() {
	var err error
	s.selectSiteStmt, err = s.db.Prepare(
		"SELECT id, name, subdomain FROM sites WHERE subdomain = ?")
	if err != nil {
		log.Println(err)
	}
}

func (s *session) prepareSelectFeedsStmt() {
	var err error
	s.selectFeedStmt, err = s.db.Prepare(
		"SELECT f.id, f.site_id, f.name, f.url, n.products_field, " +
			"n.name_field, n.identifier_field, n.description_field, n.price_field, " +
			"n.producturl_field, n.regular_price_field, n.currency_field, " +
			"n.shipping_price_field, n.in_stock_field, n.graphicurl_field, " +
			"n.categories_field, f.sync_categories, f.allow_empty_description " +
			"FROM feeds as f " +
			"INNER JOIN networks as n " +
			"ON f.`network_id` = n.`id` " +
			"WHERE f.site_id = ?")
	if err != nil {
		log.Println(err)
	}
}

func (s *session) prepareSelectCategoryStmt() {
	var err error
	s.selectCategoryStmt, err = s.db.Prepare("SELECT id, name, slug, " +
		"keywords, description_by_user, created_by_id FROM categories " +
		"WHERE site_id = ?")
	if err != nil {
		log.Println(err)
	}
}

func (s *session) prepareSelectFeedProductStmt() {
	var err error
	s.selectFeedProductStmt, err = s.db.Prepare(
		"SELECT id, site_id, feed_id, name, name_by_user, identifier, price, " +
			"regular_price, description, description_by_user, keywords, " +
			"currency, url, graphic_url, " + "shipping_price, in_stock " +
			"FROM products WHERE feed_id = ?")
	if err != nil {
		log.Println(err)
	}
}

func (s *session) prepareSelectCategoryProductStmt() {
	var err error
	s.selectCategoryProductStmt, err = s.db.Prepare(
		"SELECT cp.id, c.name, c.keywords, c.description_by_user, " +
			"cp.category_id, cp.created_by_id " +
			"FROM categories c INNER JOIN category_product AS cp " +
			"ON c.`id` = cp.`category_id` " +
			"WHERE cp.`product_id` = ?")
	if err != nil {
		log.Println(err)
	}
}

func (s *session) selectFeeds() error {
	s.feeds = []*feed{}
	rows, err := s.selectFeedStmt.Query(s.site.ID)
	if err != nil {
		log.Println(err)
	}

	defer rows.Close()
	for rows.Next() {
		f := &feed{}
		err = rows.Scan(
			&f.ID,
			&f.SiteID,
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
			&f.SyncCategories,
			&f.AllowEmptyDescription,
		)

		if err != nil {
			log.Println(err)
		} else {
			s.feeds = append(s.feeds, f)
		}
	}

	if err := rows.Err(); err != nil {
		log.Println(err)
	}
	return err
}

func (s *session) prepare() {
	s.FeedDone = make(chan feedmessage, len(s.feeds))
	s.FeedError = make(chan feedmessage, len(s.feeds))
	s.DBOperation = make(chan message)

	// Run 5 worker instances for db actions
	for i := 0; i < 5; i++ {
		go s.worker()
	}
}

func (s *session) getResult() {
	for i := 0; i < len(s.feeds); i++ {
		select {
		case m := <-s.FeedDone:
			log.Println(m.feed.Name + " " + m.action + " completed.")
		case m := <-s.FeedError:
			log.Println("Errors in "+m.feed.Name+" "+m.action, m.err)
		}
	}
}

func (s *session) syncProductCategories(update bool) {
	var err error
	s.categories, err = s.selectCategories()
	if err != nil {
		log.Print(err)
	}
	for _, f := range s.feeds {
		go f.syncProductCategories(s, update)
	}
}

func (s *session) update() {
	defer s.db.Close()

	s.cleanCategories()
	for _, f := range s.feeds {
		go f.update(s)
	}
	s.getResult()

	s.resetCategories()
	var err error
	s.categories, err = s.selectCategories()
	if err != nil {
		log.Print(err)
	}

	update := true
	s.syncProductCategories(update)
	s.getResult()
}

func (s *session) refresh() {
	defer s.db.Close()

	for _, f := range s.feeds {
		go f.refresh(s)
	}
	s.getResult()
	update := false
	s.syncProductCategories(update)
	s.getResult()
}

func (s *session) worker() {
	var wg sync.WaitGroup
	for {
		var err error
		select {
		case message := <-s.DBOperation:
			switch message.entity.getDBAction() {

			case DBACTION_INSERT:
				wg.Add(1)
				go func() {
					defer wg.Done()
					err = message.entity.insert(s)
					if err != nil {
						log.Println(err)
						message.feed.DBOperationError <- err
					} else {
						message.feed.DBOperationDone <- fmt.Sprintf(
							"Inserted %s: '%s'.",
							message.entity.getEntityType(),
							message.entity.getName())
					}
				}()

			case DBACTION_UPDATE:
				wg.Add(1)
				go func() {
					defer wg.Done()
					err = message.entity.update(s)
					if err != nil {
						log.Println(err)
						message.feed.DBOperationError <- err
					} else {
						message.feed.DBOperationDone <- fmt.Sprintf(
							"Updated %s: '%s'.",
							message.entity.getEntityType(),
							message.entity.getName())
					}
				}()

			case DBACTION_DELETE:
				wg.Add(1)
				go func() {
					defer wg.Done()
					err = message.entity.delete(s)
					if err != nil {
						log.Println(err)
						message.feed.DBOperationError <- err
					} else {
						message.feed.DBOperationDone <- fmt.Sprintf(
							"Deleted %s: '%s'.",
							message.entity.getEntityType(),
							message.entity.getName())
					}
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

func (s *session) cleanCategories() {
	s.categories, _ = s.selectCategories()

	for i := len(s.categories) - 1; i >= 0; i-- {
		c := s.categories[i]
		if c.getName() == "" {
			err := c.delete(s)
			if err != nil {
				log.Println(err)
			} else {
				log.Println(fmt.Sprintf("Deleted empty category: %v", c.getID()))
				s.categories = append(s.categories[:i], s.categories[i+1:]...)
			}
		}
	}
}

func (s *session) resetCategories() {
	s.categories = []categoryinterface{}
}

func (s *session) selectSite(subdomain string) (site, error) {
	var si site
	rows, err := s.selectSiteStmt.Query(subdomain)
	if err != nil {
		log.Println(err)
		return si, err
	}

	defer rows.Close()
	for rows.Next() {
		si = site{}
		err := rows.Scan(&si.ID, &si.Name, &si.Subdomain)
		if err != nil {
			log.Println(err)
		}
	}

	err = rows.Err()
	if err != nil {
		log.Println(err)
	}

	s.site = &si
	return si, err
}

func (s *session) selectCategories() ([]categoryinterface, error) {
	if len(s.categories) > 0 {
		return s.categories, nil
	}

	categories := []categoryinterface{}
	rows, err := s.selectCategoryStmt.Query(s.site.ID)
	if err != nil {
		log.Println(err)
		return categories, err
	}

	defer rows.Close()
	for rows.Next() {
		c := category{}
		err := rows.Scan(&c.ID, &c.Name, &c.Slug, &c.Keywords, &c.DescriptionByUser,
			&c.CreatedByID)
		if err != nil {
			log.Println(err)
		} else {
			categories = append(categories, &c)
		}
	}

	err = rows.Err()
	if err != nil {
		log.Println(err)
	}

	s.categories = categories
	return categories, err
}
