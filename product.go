package main

import (
	"database/sql"
	"log"
	"strings"
	"unicode"
)

type product struct {
	ID                int
	SiteID            int
	FeedID            int
	BrandID           *int64
	Name              string
	NameByUser        string
	Slug              string
	Identifier        string
	Categories        []categoryinterface
	Description       string
	DescriptionByUser string
	Brand             string
	Price             string
	ProductURL        string
	GraphicURL        string
	RegularPrice      string
	Currency          string
	ShippingPrice     string
	InStock           bool
	Points            int
	HasCategories     bool
	Active            bool
	DBAction          int
	CreatedAt         string
	UpdatedAt         string
	DeletedAt         sql.NullString
}

func (p product) getName() string {
	if p.NameByUser != "" {
		return p.NameByUser
	}
	return p.Name
}

func (p product) setSiteID(siteID int) {

}

func (p product) getEntityType() string {
	return "product"
}

func (p product) getDBAction() int {
	return p.DBAction
}

func (p product) insert(s *session) error {
	_, err := s.db.Exec(
		"INSERT INTO products (name, site_id, slug, feed_id, identifier, description, "+
			"price, regular_price, currency, shipping_price, "+
			"in_stock, url, graphic_url, created_at, updated_at) "+
			"VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,now(),now())",
		p.Name,
		p.SiteID,
		p.Slug,
		p.FeedID,
		p.Identifier,
		p.Description,
		p.Price,
		p.RegularPrice,
		p.Currency,
		p.ShippingPrice,
		p.InStock,
		p.ProductURL,
		p.GraphicURL,
	)
	return err
}

func (p product) update(s *session) error {
	_, err := s.db.Exec(
		"UPDATE products SET name = ?, identifier = ?, description = ?, "+
			"price = ?, regular_price = ?, currency = ?, shipping_price = ?,"+
			"in_stock = ?, url = ?, graphic_url = ?, has_categories = ?, "+
			"updated_at = now(), deleted_at = ? WHERE id = ?",
		p.Name,
		p.Identifier,
		p.Description,
		p.Price,
		p.RegularPrice,
		p.Currency,
		p.ShippingPrice,
		p.InStock,
		p.ProductURL,
		p.GraphicURL,
		p.HasCategories,
		p.DeletedAt,
		p.ID,
	)
	return err
}

func (p *product) updateHasCategories(s *session) error {
	var count int
	err := s.selectCategoryCountByProductIDStmt.QueryRow(p.ID).Scan(&count)

	if err != nil {
		log.Println(err)
	}

	if count == 0 && p.HasCategories == true {
		p.HasCategories = false
		p.update(s)
	} else if count > 0 && p.HasCategories == false {
		p.HasCategories = true
		p.update(s)
	}
	return err
}

func (p product) delete(s *session) error {
	_, err := s.db.Exec("UPDATE products SET deleted_at = NOW() WHERE id = ?", p.ID)

	return err
}

func (p *product) resetCategories() {
	p.Categories = []categoryinterface{}
}

func (p *product) selectCategories(s *session) ([]categoryinterface, error) {
	categoryproducts := []categoryinterface{}
	rows, err := s.selectCategoryProductStmt.Query(p.ID)

	if err != nil {
		log.Println(err)
		return categoryproducts, err
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
			&cp.CategoryID,
			&cp.product.ID,
			&cp.Forced,
		)
		if err != nil {
			log.Println(err)
			return categoryproducts, err
		} else {
			categoryproducts = append(categoryproducts, &cp)
		}
	}

	err = rows.Err()
	return categoryproducts, err
}

func (p *product) selectCategoryProduct(s *session, c *category) (categoryproduct, error) {
	var cp categoryproduct
	rows, err := s.selectCategoryProductByProductIDAndCategoryIDStmt.Query(p.ID, c.ID)

	if err != nil {
		log.Println(err)
		return cp, err
	}

	defer rows.Close()
	for rows.Next() {
		cp = categoryproduct{}
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
			&cp.CategoryID,
			&cp.product.ID,
			&cp.Forced,
		)
		if err != nil {
			log.Println(err)
		}
	}

	err = rows.Err()
	return cp, err
}

func (p *product) attachCategory(s *session, c categoryinterface) error {
	_, err := s.db.Exec(
		"INSERT INTO category_product "+
			"(category_id, product_id, created_at,"+
			" updated_at) VALUES (?,?,now(),now())",
		c.getCategoryID(),
		p.ID,
	)

	err = p.updateHasCategories(s)

	if err != nil {
		log.Println("Could not attach category "+c.getName()+" to: "+p.Name, err)
	} else {
		log.Println("Attached category " + c.getName() + " to: " + p.Name)
	}
	return err
}

func (p *product) detachCategory(s *session, cp categoryproduct) error {
	_, err := s.db.Exec(
		"DELETE FROM category_product "+
			"WHERE id = ?", cp.ID)

	err = p.updateHasCategories(s)

	if err != nil {
		log.Println("Could not detach category "+cp.category.Name+" from: "+p.Name, err)
	} else {
		log.Println("Detached category " + cp.category.Name + " from: " + p.Name)
	}
	return err
}

func (p *product) validate(f *feed) bool {
	if f.AllowEmptyDescription == false {
		if p.Description == "" {
			return false
		}
	}
	return true
}

func generateSlug(str string) (slug string) {
	return strings.Map(func(r rune) rune {
		switch {
		case r == ' ', r == '-':
			return '-'
		case r == '_', unicode.IsLetter(r), unicode.IsDigit(r):
			return r
		default:
			return -1
		}
		return -1
	}, strings.ToLower(strings.TrimSpace(str)))
}

func (p *product) indexesOf(slice []categoryproduct) []int {
	indexes := []int{}
	for i, ele := range slice {
		if ele.product.ID == p.ID {
			indexes = append(indexes, i)
		}
	}
	return indexes
}
