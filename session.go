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

type session struct {
	db                                                *sql.DB
	selectSiteStmt                                    *sql.Stmt
	selectFeedStmt                                    *sql.Stmt
	insertProductStmt                                 *sql.Stmt
	selectCategoryStmt                                *sql.Stmt
	insertCategoryStmt                                *sql.Stmt
	selectFeedProductsStmt                            *sql.Stmt
	selectFeedNetworkStmt                             *sql.Stmt
	deleteProductStmt                                 *sql.Stmt
	selectCategoryProductStmt                         *sql.Stmt
	selectCategoryProductsByCategoryIDStmt            *sql.Stmt
	selectCategoryProductByProductIDAndCategoryIDStmt *sql.Stmt
	selectCategoryProductByCategoryProductIDStmt      *sql.Stmt
	selectCategoryCountByProductIDStmt                *sql.Stmt
	insertCategoryProductStmt                         *sql.Stmt
	searchCategoryProductsStmt                        *sql.Stmt
	deleteCategoryProductStmt                         *sql.Stmt
	site                                              *site
	feeds                                             []*feed
	categories                                        []categoryinterface
	DBOperation                                       chan message
	FeedDone                                          chan feedmessage
	FeedError                                         chan feedmessage
	CategoryDone																			chan categorymessage
}

func (s *session) init(subdomain string) error {
	var err error
	// This does not really open a new connection.
	var DSN = fmt.Sprintf("%v:%v@tcp(%v:%v)/%v", *dbUser, *dbPassword, *dbAddr, *dbPort, *database)
	s.db, err = sql.Open("mysql", DSN)
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
		s.prepareSearchCategoryProductsStmt()
		s.prepareSelectFeedProductsStmt()
		s.prepareSelectFeedNetworkStmt()
		s.prepareSelectCategoryProductStmt()
		s.prepareSelectCategoryCountByProductIDStmt()
		s.prepareSelectCategoryProductByProductIDAndCategoryIDStmt()
		s.prepareSelectCategoryProductsByCategoryIDStmt()
		s.prepareSelectCategoryProductByCategoryProductIDStmt()
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
		"SELECT f.id, f.site_id, f.name, f.url, f.network_id, " +
			"f.allow_empty_description " +
			"FROM feeds as f " +
			"WHERE f.site_id = ?")
	if err != nil {
		log.Println(err)
	}
}

func (s *session) prepareSelectCategoryStmt() {
	var err error
	s.selectCategoryStmt, err = s.db.Prepare("SELECT id, name, slug, " +
		"search, description FROM categories " +
		"WHERE site_id = ?")
	if err != nil {
		log.Println(err)
	}
}

func (s *session) prepareSelectCategoryCountByProductIDStmt() {
	var err error
	s.selectCategoryCountByProductIDStmt, err = s.db.Prepare("SELECT COUNT(*) " +
		"FROM category_product " +
		"WHERE product_id = ?")
	if err != nil {
		log.Println(err)
	}
}

func (s *session) prepareSearchCategoryProductsStmt() {
	var err error
	s.searchCategoryProductsStmt, err = s.db.Prepare("SELECT * FROM products " +
		"WHERE site_id = ? " +
		"AND MATCH(`name`,`description`) " +
		"AGAINST (? IN BOOLEAN MODE)")
	if err != nil {
		log.Println(err)
	}
}

func (s *session) prepareSelectFeedProductsStmt() {
	var err error
	s.selectFeedProductsStmt, err = s.db.Prepare(
		"SELECT id, site_id, feed_id, name, name_by_user, identifier, price, " +
			"regular_price, description, description_by_user, " +
			"currency, url, graphic_url, shipping_price, in_stock, " +
			"points, has_categories, active, deleted_at " +
			"FROM products WHERE feed_id = ?")
	if err != nil {
		log.Println(err)
	}
}

func (s *session) prepareSelectFeedNetworkStmt() {
	var err error
	s.selectFeedNetworkStmt, err = s.db.Prepare(
		"SELECT id, name FROM networks WHERE id = ? LIMIT 1")
	if err != nil {
		log.Println(err)
	}
}

func (s *session) prepareSelectCategoryProductsByCategoryIDStmt() {
	var err error
	s.selectCategoryProductsByCategoryIDStmt, err = s.db.Prepare(
		"SELECT cp.id, p.site_id, p.feed_id, p.name, p.name_by_user, p.identifier, p.price, " +
			"p.regular_price, p.description, p.description_by_user, " +
			"p.currency, p.url, p.graphic_url, p.shipping_price, p.in_stock, " +
			"p.points, p.has_categories, p.active, p.deleted_at, " +
			"cp.category_id, cp.product_id, cp.forced " +
			"FROM products p " +
			"INNER JOIN category_product cp " +
			"ON p.id = cp.product_id " +
			"WHERE cp.category_id = ?")
	if err != nil {
		log.Println(err)
	}
}

func (s *session) prepareSelectCategoryProductByCategoryProductIDStmt() {
	var err error
	s.selectCategoryProductByCategoryProductIDStmt, err = s.db.Prepare(
		"SELECT cp.id, p.site_id, p.feed_id, p.name, p.name_by_user, p.identifier, p.price, " +
			"p.regular_price, p.description, p.description_by_user, " +
			"p.currency, p.url, p.graphic_url, p.shipping_price, p.in_stock, " +
			"p.points, p.has_categories, p.active, " +
			"cp.category_id, cp.product_id, cp.forced " +
			"FROM products p " +
			"INNER JOIN category_product cp " +
			"ON p.id = cp.product_id " +
			"WHERE cp.id = ?")
	if err != nil {
		log.Println(err)
	}
}

func (s *session) prepareSelectCategoryProductStmt() {
	var err error
	s.selectCategoryProductStmt, err = s.db.Prepare(
		"SELECT cp.id, c.name, c.search, c.description, " +
			"cp.category_id, cp.forced " +
			"FROM categories c INNER JOIN category_product AS cp " +
			"ON c.`id` = cp.`category_id` " +
			"WHERE cp.`product_id` = ?")
	if err != nil {
		log.Println(err)
	}
}

func (s *session) prepareSelectCategoryProductByProductIDAndCategoryIDStmt() {
	var err error
	s.selectCategoryProductByProductIDAndCategoryIDStmt, err = s.db.Prepare(
		"SELECT cp.id, p.site_id, p.feed_id, p.name, p.name_by_user, p.identifier, p.price, " +
			"p.regular_price, p.description, p.description_by_user, " +
			"p.currency, p.url, p.graphic_url, p.shipping_price, p.in_stock, " +
			"p.points, p.has_categories, p.active, " +
			"cp.category_id, cp.product_id, cp.forced " +
			"FROM products p " +
			"INNER JOIN category_product cp " +
			"ON p.id = cp.product_id " +
			"WHERE p.id = ? AND cp.category_id = ?")
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
			&f.NetworkID,
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

func (s *session) waitForResult() {
	for i := 0; i < len(s.feeds); i++ {
		select {
		case m := <-s.FeedDone:
			log.Println(m.feed.Name + " " + m.action + " completed.")
			s.syncProductCategories()
			s.waitForRefreshResult()
		case m := <-s.FeedError:
			log.Println("Errors in "+m.feed.Name+" "+m.action, m.err)
			<-SessionQueue
		}
	}
}

func (s *session) waitForRefreshResult() {
	for i := 0; i < len(s.categories); i++ {
		select {
		case m := <-s.CategoryDone:
			log.Println(m.category.Name + " completed.")
		}
		if i == len(s.categories) - 1 {
			log.Println("Session " + s.site.Name + " done")
 			<-SessionQueue
    }
	}
}

func (s *session) syncProductCategories() {
	var err error
	s.categories, err = s.selectCategories()
	s.CategoryDone = make(chan categorymessage, len(s.categories))
	if err != nil {
		log.Print(err)
	}

	for _, c := range s.categories {
		err = c.syncProducts(s)
		if err != nil {
			log.Print(err)
		}

	}
}

func (s *session) update() {
	defer s.db.Close()

	for _, f := range s.feeds {
		var err error
		f.Network, err = f.selectNetwork(s)
		if err != nil {
			log.Println(err)
		} else {
			go f.update(s)
		}
	}
	s.waitForResult()
}

func (s *session) refresh() {
	defer s.db.Close()

	s.syncProductCategories()
	s.waitForRefreshResult()
}

func (s *session) worker() {
	var wg sync.WaitGroup
	for {
		var err error
		select {
		case message := <-s.DBOperation:
			switch message.product.getDBAction() {
			case DBACTION_INSERT:
				wg.Add(1)
				go func() {
					defer wg.Done()
					err = message.product.insert(s)
					if err != nil {
						log.Println(err)
						message.feed.DBOperationError <- err
					} else {
						message.feed.DBOperationDone <- fmt.Sprintf(
							"Inserted %s: '%s'.",
							message.product.getEntityType(),
							message.product.getName())
					}
				}()

			case DBACTION_UPDATE:
				wg.Add(1)
				go func() {
					defer wg.Done()
					err = message.product.update(s)
					if err != nil {
						log.Println(err)
						message.feed.DBOperationError <- err
					} else {
						message.feed.DBOperationDone <- fmt.Sprintf(
							"Updated %s: '%s'.",
							message.product.getEntityType(),
							message.product.getName())
					}
				}()

			case DBACTION_DELETE:
				wg.Add(1)
				go func() {
					defer wg.Done()
					err = message.product.delete(s)
					if err != nil {
						log.Println(err)
						message.feed.DBOperationError <- err
					} else {
						message.feed.DBOperationDone <- fmt.Sprintf(
							"Deleted %s: '%s'.",
							message.product.getEntityType(),
							message.product.getName())
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
		err := rows.Scan(
			&c.ID,
			&c.Name,
			&c.Slug,
			&c.Search,
			&c.Description,
		)
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
