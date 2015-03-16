package main

import (
	"fmt"
	"log"
	"strings"
	"unicode"
)

type product struct {
	ID            int
	FeedID        int64
	Name          string
	Slug          string
	Identifier    string
	Categories    []category
	Description   string
	Brand         string
	Price         string
	ProductURL    string
	GraphicURL    string
	RegularPrice  string
	Currency      string
	ShippingPrice string
	InStock       int
	DBAction      int
}

func (p product) getDBAction() int {
	return p.DBAction
}

func (p product) insert(f *feed, s *session) {
	result, err := s.db.Exec(
		"INSERT INTO products (name, slug, feed_id, identifier, description, "+
			"price, regular_price, currency, shipping_price, in_stock, url,"+
			"graphic_url, created_at, updated_at) "+
			"VALUES (?,?,?,?,?,?,?,?,?,?,?,?,now(),now())",
		p.Name,
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

	if err != nil {
		f.DBOperationError <- err
	} else {
		id, err := result.LastInsertId()
		p.ID = int(id)

		err = p.updateCategories(s)

		if err != nil {
			f.DBOperationError <- err
		} else {
			f.DBOperationDone <- fmt.Sprintf("New product: '%v'.", p.Name)
		}
	}
}

func (p product) update(f *feed, s *session) {
	_, err := s.db.Exec(
		"UPDATE products SET name = ?, identifier = ?, description = ?, "+
			"price = ?, regular_price = ?, currency = ?, shipping_price = ?,"+
			"in_stock = ?, url = ?, graphic_url = ?, updated_at = now() "+
			"WHERE id = ?",
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
		p.ID,
	)

	if err != nil {
		f.DBOperationError <- err
	} else {

		err = p.updateCategories(s)

		if err != nil {
			f.DBOperationError <- err
		} else {
			f.DBOperationDone <- fmt.Sprintf("Updated product: '%v'.", p.Name)
		}
	}
}

func (p product) delete(f *feed, s *session) {
	_, err := s.db.Exec("DELETE FROM products WHERE id = ?", p.ID)

	if err != nil {
		f.DBOperationError <- err
	} else {
		f.DBOperationDone <- fmt.Sprintf("Deleted product: '%v'.", p.Name)
	}
}

func (p product) selectCategories(s *session) ([]category, error) {
	categories := []category{}
	rows, err := s.selectCategoryProductStmt.Query(p.ID)

	if err != nil {
		return categories, err
	}

	defer rows.Close()
	for rows.Next() {
		c := category{}
		err := rows.Scan(&c.ID, &c.Name, &c.CategoryProductID)
		if err != nil {
			return categories, err
		} else {
			categories = append(categories, c)
		}
	}

	err = rows.Err()
	return categories, err
}

func (p *product) updateCategories(s *session) error {
	productCategories, err := p.selectCategories(s)
	if err != nil {
		return err
	}

	categories, err := s.selectCategories()
	if err != nil {
		return err
	}

	for _, c := range productCategories {
		// If db category is not in feed categories, detach it.
		indexes := c.indexesOf(p.Categories)
		if len(indexes) == 0 {
			for _, v := range categories {
				if c.Name == v.Name {
					c.ID = v.ID
					err := p.detachCategory(s, c)
					if err != nil {
						return err
					}
				}
			}
		}
	}

	for _, c := range p.Categories {
		indexes := c.indexesOf(productCategories)
		// If feed category is not in db, attach it.
		if len(indexes) == 0 {
			for _, v := range categories {
				if c.Name == v.Name {
					c.ID = v.ID
				}
			}

			err := p.attachCategory(s, c)
			if err != nil {
				return err
			}
		}

		// If more than one occurrence in db, remove the rest.
		if len(indexes) > 1 {
			for _, v := range indexes[1:] {
				c = productCategories[v]
				err := p.detachCategory(s, c)
				if err != nil {
					return err
				}
			}
		}
	}

	return err

}

func (p product) identicalCategories(s *session, feedCategories []category) (bool, error) {
	productCategories, err := p.selectCategories(s)
	if err != nil {
		return false, err
	}

	for _, c := range productCategories {
		indexes := c.indexesOf(feedCategories)
		if len(indexes) != 1 {
			return false, nil
		}
	}

	for _, c := range feedCategories {
		indexes := c.indexesOf(productCategories)
		if len(indexes) != 1 {
			return false, nil
		}
	}

	return true, nil
}

func (p product) attachCategory(s *session, c category) error {
	_, err := s.db.Exec(
		"INSERT INTO category_product "+
			"(category_id, product_id, created_at, updated_at) "+
			"VALUES (?,?,now(),now())",
		c.ID,
		p.ID,
	)

	if err != nil {
		return err
	}

	if err != nil {
		log.Println("Could not attach category " + c.Name + " to: " + p.Name)
	} else {
		log.Println("Attached category " + c.Name + " to: " + p.Name)
	}
	return err
}

func (p product) detachCategory(s *session, c category) error {
	_, err := s.db.Exec(
		"DELETE FROM category_product "+
			"WHERE id = ?", c.CategoryProductID)

	if err != nil {
		log.Println("Could not detach category " + c.Name + " from: " + p.Name)
	} else {
		log.Println("Detached category " + c.Name + " from: " + p.Name)
	}
	return err
}

// parseProduct extracts a product struct from an interface
func (p *product) parse(i interface{}, f *feed) {

	name, ok := i.(map[string]interface{})[f.NameField].(string)
	if ok {
		p.Name = name
	}

	p.Slug = generateSlug(p.Name)

	identifier, ok := i.(map[string]interface{})[f.IdentifierField].(string)
	if ok {
		p.Identifier = identifier
	}

	price, ok := i.(map[string]interface{})[f.PriceField].(string)
	if ok {
		p.Price = price
	}

	regularPrice, ok := i.(map[string]interface{})[f.RegularPriceField].(string)
	if ok {
		p.RegularPrice = regularPrice
	}

	description, ok := i.(map[string]interface{})[f.DescriptionField].(string)
	if ok {
		p.Description = description
	}

	currency, ok := i.(map[string]interface{})[f.CurrencyField].(string)
	if ok {
		p.Currency = currency
	}

	productUrl, ok := i.(map[string]interface{})[f.ProductURLField].(string)
	if ok {
		p.ProductURL = productUrl
	}

	graphicURL, ok := i.(map[string]interface{})[f.GraphicURLField].(string)
	if ok {
		p.GraphicURL = graphicURL
	}

	shippingPrice, ok := i.(map[string]interface{})[f.ShippingPriceField].(string)
	if ok {
		p.ShippingPrice = shippingPrice
	}

	inStock, ok := i.(map[string]interface{})[f.InStockField].(int)
	if ok {
		p.InStock = inStock
	}

	p.Categories = p.parseCategories(i, f)
}

func (p product) parseCategories(i interface{}, f *feed) []category {
	switch v := i.(map[string]interface{})[f.CategoriesField].(type) {
	case []interface{}:
		return categoriesFromList(v, f)
	case string:
		return categoriesFromString(v, f)
	}
	return make([]category, 0)
}

// Appends to feed categories if not included already
func categoriesFromList(list []interface{}, f *feed) []category {
	if len(list) > 0 {
		categories := make([]category, len(list))
		for i, v := range list {
			c := category{Name: v.(string)}
			categories[i] = c
			f.Categories = c.appendIfMissing(f.Categories)
		}
		return categories
	}
	return make([]category, 0)
}

// parseCategoriesFromString creates a string array from string.
// Appends to feed categories if not included already
func categoriesFromString(s string, f *feed) []category {
	c := category{Name: s}
	categories := make([]category, 1)
	categories[0] = c
	f.Categories = c.appendIfMissing(f.Categories)
	return categories
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
