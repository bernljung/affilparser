package main

import "log"

type categorymessage struct {
	category   *category
	err    error
	action string
}

type category struct {
	ID          int
	ParentID    int
	Name        string
	Slug        string
	SiteID      int
	Search      string
	Description string
	CreatedByID int
	DBAction    int
}

func (c *category) getName() string {
	return c.Name
}

func (c category) getEntityType() string {
	return "category"
}

func (c *category) setSiteID(siteID int) {
	c.SiteID = siteID
}

func (c *category) getDescription() string {
	return c.Description
}

func (c *category) getSearchString() string {
	return c.Search
}

func (c *category) getCategoryID() int {
	return c.ID
}

func (c *category) getCreatedByID() int {
	return c.CreatedByID
}

func (c *category) getDBAction() int {
	return c.DBAction
}

func (c *category) setDBAction(action int) {
	c.DBAction = action
}

func (c *category) setCreatedByID(createdById int) {
	c.CreatedByID = createdById
}

func (c *category) getID() int {
	return c.ID
}

func (c *category) setCategoryID(id int) {
	c.ID = id
}

func (c *category) selectProducts(s *session) ([]categoryproduct, error) {
	var categoryProducts []categoryproduct
	rows, err := s.selectCategoryProductsByCategoryIDStmt.Query(c.ID)

	if err != nil {
		log.Println(err)
		return categoryProducts, err
	}

	defer rows.Close()
	for rows.Next() {
		cp := categoryproduct{}
		err := rows.Scan(
			&cp.ID,
			&cp.product.SiteID,
			&cp.product.FeedID,
			&cp.product.Name,
			&cp.product.NameByUser,
			&cp.product.Identifier,
			&cp.product.Price,
			&cp.product.RegularPrice,
			&cp.product.Description,
			&cp.product.DescriptionByUser,
			&cp.product.Currency,
			&cp.product.ProductURL,
			&cp.product.GraphicURL,
			&cp.product.ShippingPrice,
			&cp.product.InStock,
			&cp.product.Points,
			&cp.product.HasCategories,
			&cp.product.Active,
			&cp.product.DeletedAt,
			&cp.category.ID,
			&cp.product.ID,
			&cp.Forced,
		)
		if err != nil {
			log.Println(err)
			return categoryProducts, err
		} else {
			categoryProducts = append(categoryProducts, cp)
		}
	}

	err = rows.Err()
	return categoryProducts, err
}

func (c *category) insert(s *session) error {
	_, err := s.db.Exec(
		"INSERT INTO categories (name, slug, site_id, created_at, updated_at) "+
			"VALUES (?,?,?,now(),now())",
		c.Name,
		c.Slug,
		c.SiteID,
		false,
	)
	return err
}

func (c *category) update(s *session) error {
	var err error
	return err
}

func (c *category) delete(s *session) error {
	_, err := s.db.Exec("UPDATE categories SET deleted_at = NOW() WHERE id = ?", c.ID)
	return err
}

func (c *category) syncProducts(s *session) error {
	var err error

	activeProducts, err := c.selectProducts(s)
	if err != nil {
		log.Println(err)
	}

	searchProducts := []product{}
	log.Println(s.site.ID, c.Search)
	rows, err := s.searchCategoryProductsStmt.Query(s.site.ID, c.Search)
	if err != nil {
		log.Println(err)
	} else {
		defer rows.Close()
		for rows.Next() {
			p := product{}
			err := rows.Scan(
				&p.ID,
				&p.SiteID,
				&p.FeedID,
				&p.BrandID,
				&p.NameByUser,
				&p.Name,
				&p.Slug,
				&p.Identifier,
				&p.Price,
				&p.RegularPrice,
				&p.DescriptionByUser,
				&p.Description,
				&p.Currency,
				&p.ProductURL,
				&p.GraphicURL,
				&p.ShippingPrice,
				&p.InStock,
				&p.Points,
				&p.HasCategories,
				&p.Active,
				&p.CreatedAt,
				&p.UpdatedAt,
				&p.DeletedAt,
			)

			if err != nil {
				log.Println(err)
			} else {
				searchProducts = append(searchProducts, p)
			}
		}

		for _, p := range searchProducts {
			indexes := p.indexesOf(activeProducts)
			if len(indexes) == 0 {
				p.attachCategory(s, c)
			}
		}

		for _, p := range activeProducts {
			indexes := []int{}
			for i, ele := range searchProducts {
				if ele.ID == p.product.ID {
					indexes = append(indexes, i)
				}
			}
			if len(indexes) == 0 {
				cp, err := p.selectCategoryProduct(s, c)
				if err != nil {
					log.Println(err)
				}

				if cp.Forced == false {
					p.detachCategory(s, cp)
				}
			}
		}
	}
	
	s.CategoryDone <- categorymessage{category: c, err: nil, action: "syncProducts"}
	return err
}
