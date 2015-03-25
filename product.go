package main

import (
	"log"
	"regexp"
	"strings"
	"unicode"
)

type product struct {
	ID                int
	SiteID            int
	FeedID            int
	Name              string
	NameByUser        string
	Slug              string
	Identifier        string
	FeedCategories    []categoryinterface
	Categories        []categoryinterface
	Description       string
	DescriptionByUser string
	Brand             string
	Price             string
	ProductURL        string
	GraphicURL        string
	RegularPrice      string
	Keywords          string
	Currency          string
	ShippingPrice     string
	InStock           bool
	DBAction          int
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
			"in_stock, url, graphic_url, keywords, created_at, updated_at) "+
			"VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,now(),now())",
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
		p.Keywords,
	)
	return err
}

func (p product) update(s *session) error {
	log.Println("Insert: ", p.getName())
	_, err := s.db.Exec(
		"UPDATE products SET name = ?, identifier = ?, description = ?, "+
			"price = ?, regular_price = ?, currency = ?, shipping_price = ?,"+
			"in_stock = ?, url = ?, graphic_url = ?, keywords = ?, "+
			"updated_at = now() WHERE id = ?",
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
		p.Keywords,
		p.ID,
	)

	return err
}

func (p product) delete(s *session) error {
	_, err := s.db.Exec("DELETE FROM products WHERE id = ?", p.ID)

	return err
}

func (p *product) resetCategories() {
	p.Categories = []categoryinterface{}
}

func (p *product) selectCategories(s *session) ([]categoryinterface, error) {
	if len(p.Categories) != 0 {
		return p.Categories, nil
	}

	categoryproducts := []categoryinterface{}
	rows, err := s.selectCategoryProductStmt.Query(p.ID)

	if err != nil {
		log.Println(err)
		return categoryproducts, err
	}

	defer rows.Close()
	for rows.Next() {
		c := categoryproduct{}
		err := rows.Scan(
			&c.ID,
			&c.Name,
			&c.Keywords,
			&c.DescriptionByUser,
			&c.CategoryID,
			&c.CreatedByID,
		)
		if err != nil {
			log.Println(err)
			return categoryproducts, err
		} else {
			categoryproducts = append(categoryproducts, &c)
		}
	}

	err = rows.Err()
	return categoryproducts, err
}

func (p *product) getOutdatedFeedCategories(s *session, f *feed) []categoryinterface {
	outdatedCategories := []categoryinterface{}
	for _, c := range p.Categories {
		indexes := c.indexesOf(f.Products[p.Identifier].FeedCategories)
		if len(indexes) == 0 {
			outdatedCategories = append(outdatedCategories, c)
		}
	}
	return outdatedCategories
}

func (p *product) getFeedCategories(s *session, f *feed) []categoryinterface {
	newCategories := []categoryinterface{}
	for _, c := range f.Products[p.Identifier].FeedCategories {
		for _, v := range s.categories {
			if c.getName() == v.getName() {
				c.setCategoryID(v.getCategoryID())
				newCategories = append(newCategories, c)
			}
		}
	}
	return newCategories
}

func (p *product) getDuplicateCategories(s *session) []categoryinterface {
	duplicateCategories := []categoryinterface{}
	for _, c := range p.Categories {
		indexes := c.indexesOf(p.Categories)
		if len(indexes) > 1 {
			for _, v := range indexes[1:] {
				c := p.Categories[v]
				duplicateCategories = c.appendIfMissing(duplicateCategories)
			}
		}
	}
	return duplicateCategories
}

func (p *product) syncCategories(s *session, f *feed, update bool) error {
	var err error
	p.Categories, err = p.selectCategories(s)
	if err != nil {
		log.Println(err)
		return err
	}

	categoriesToAttach := []categoryinterface{}
	categoriesToDetach, err := p.selectCategories(s)
	keywordCategories := p.getKeywordCategories(s)

	if update == true {

		outdatedFeedCategories := []categoryinterface{}
		feedCategories := []categoryinterface{}

		if f.SyncCategories {
			outdatedFeedCategories = p.getOutdatedFeedCategories(s, f)
			feedCategories = p.getFeedCategories(s, f)
		}

		for _, c := range feedCategories {
			indexes := c.indexesOf(p.Categories)
			if len(indexes) == 0 {
				c.setCreatedByID(CREATED_BY_FEED)
				categoriesToAttach = c.appendIfMissing(categoriesToAttach)
			}
			categoriesToDetach = c.removeIfPresent(categoriesToDetach)
		}

		for _, c := range append(outdatedFeedCategories, p.FeedCategories...) {
			indexes := c.indexesOf(keywordCategories)
			if len(indexes) == 0 {
				categoriesToDetach = c.appendIfMissing(categoriesToDetach)
			}
		}
	}

	for _, c := range keywordCategories {
		indexes := c.indexesOf(p.Categories)
		if len(indexes) == 0 {
			c.setCreatedByID(CREATED_BY_KEYWORD)
			categoriesToAttach = c.appendIfMissing(categoriesToAttach)
		}
		categoriesToDetach = c.removeIfPresent(categoriesToDetach)
	}

	for _, c := range categoriesToAttach {
		categoriesToDetach = c.removeIfPresent(categoriesToDetach)
		p.attachCategory(s, c)
	}

	for _, c := range categoriesToDetach {

		if update == true {
			if c.getCreatedByID() != CREATED_BY_USER {
				p.detachCategory(s, c)
			}
		} else {
			if c.getCreatedByID() == CREATED_BY_KEYWORD {
				p.detachCategory(s, c)
			}
		}
	}

	p.resetCategories()
	p.Categories, err = p.selectCategories(s)
	if err != nil {
		log.Println(err)
		return err
	}
	duplicateCategories := p.getDuplicateCategories(s)

	for _, c := range duplicateCategories {
		p.detachCategory(s, c)
	}

	return err
}

func (p *product) getKeywordCategories(s *session) []categoryinterface {
	keywordCategories := []categoryinterface{}
	for _, c := range s.categories {
		if c.getKeywords() != "" {
			for _, kw := range strings.Split(c.getKeywords(), ",") {
				for _, pkw := range strings.Split(p.Keywords, ",") {
					if strings.TrimSpace(pkw) == strings.TrimSpace(kw) {
						keywordCategories = c.appendIfMissing(keywordCategories)
						continue
					}
				}
			}
		}
	}
	return keywordCategories
}

func (p *product) getNewKeywordCategories(s *session) []categoryinterface {
	keywordCategories := p.getKeywordCategories(s)
	newKeywordCategories := []categoryinterface{}

	for _, c := range keywordCategories {
		indexes := c.indexesOf(p.Categories)
		if len(indexes) == 0 {
			newKeywordCategories = append(newKeywordCategories, c)
		}
		continue
	}

	return newKeywordCategories
}

func (p *product) attachCategory(s *session, c categoryinterface) error {
	_, err := s.db.Exec(
		"INSERT INTO category_product "+
			"(category_id, product_id, created_by_id, created_at,"+
			" updated_at) VALUES (?,?,?,now(),now())",
		c.getCategoryID(),
		p.ID,
		c.getCreatedByID(),
	)

	if err != nil {
		log.Println("Could not attach category "+c.getName()+" to: "+p.Name, err)
	} else {
		log.Println("Attached category " + c.getName() + " to: " + p.Name)
	}
	return err
}

func (p *product) detachCategory(s *session, c categoryinterface) error {
	_, err := s.db.Exec(
		"DELETE FROM category_product "+
			"WHERE id = ?", c.getID())

	if err != nil {
		log.Println("Could not detach category "+c.getName()+" from: "+p.Name, err)
	} else {
		log.Println("Detached category " + c.getName() + " from: " + p.Name)
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

	inStock, ok := i.(map[string]interface{})[f.InStockField].(bool)
	if ok {
		p.InStock = inStock
	}

	p.SiteID = f.SiteID
	p.FeedID = f.ID
	p.FeedCategories = p.parseCategories(i, f)
}

func (p *product) validate(f *feed) bool {
	if f.AllowEmptyDescription == false {
		if p.Description == "" {
			return false
		}
	}
	return true
}

func (p *product) refresh() bool {
	re := regexp.MustCompile("[^a-zA-Zåäö ]")
	words := p.Description + " " + p.DescriptionByUser
	words = strings.ToLower(words)
	words = re.ReplaceAllString(words, " ")

	wordsArray := strings.Split(words, " ")
	keywords := []string{}

	for _, w := range wordsArray {
		if w != "" {
			exists := stringInSlice(w, keywords)
			if exists == false {
				keywords = append(keywords, w)
			}
		}
	}

	if p.Keywords != strings.Join(keywords, ",") {
		p.Keywords = strings.Join(keywords, ",")
		return true
	}
	return false
}

func (p *product) parseCategories(i interface{}, f *feed) []categoryinterface {
	switch v := i.(map[string]interface{})[f.CategoriesField].(type) {
	case []interface{}:
		return categoriesFromList(v, f)
	case string:
		return categoriesFromString(v, f)
	}
	return make([]categoryinterface, 0)
}

// Appends to feed categories if not included already
func categoriesFromList(list []interface{}, f *feed) []categoryinterface {
	if len(list) > 0 {
		categories := make([]categoryinterface, len(list))
		for i, v := range list {
			c := &category{Name: v.(string)}
			categories[i] = c
			f.Categories = c.appendIfMissing(f.Categories)
		}
		return categories
	}
	return make([]categoryinterface, 0)
}

// parseCategoriesFromString creates a string array from string.
// Appends to feed categories if not included already
func categoriesFromString(s string, f *feed) []categoryinterface {
	c := &category{Name: s}
	categories := make([]categoryinterface, 1)
	categories[0] = c
	f.Categories = c.appendIfMissing(f.Categories)
	return categories
}

func stringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
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
