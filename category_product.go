package main

import "log"

type categoryproduct struct {
	category
	product
	ID         int
	CategoryID int
	ProductID  int
	Forced     bool
}

func (c *categoryproduct) getName() string {
	return c.category.Name
}

func (c *categoryproduct) setSiteID(siteID int) {

}

func (c *categoryproduct) getEntityType() string {
	return "categoryproduct"
}

func (c *categoryproduct) getSearchString() string {
	return c.Search
}

func (c *categoryproduct) getDescription() string {
	return c.category.Description
}

func (c *categoryproduct) getCategoryID() int {
	return c.CategoryID
}

func (c *categoryproduct) getCreatedByID() int {
	return c.CreatedByID
}

func (c *categoryproduct) setCreatedByID(createdById int) {
	c.CreatedByID = createdById
}

func (c *categoryproduct) getDBAction() int {
	return 0
}

func (c *categoryproduct) setDBAction(action int) {
	c.category.DBAction = action
}

func (c *categoryproduct) getID() int {
	return c.ID
}

func (c *categoryproduct) setCategoryID(id int) {
	c.CategoryID = id
}

func (c *categoryproduct) insert(s *session) error {
	var err error
	return err
}
func (c *categoryproduct) update(s *session) error {
	var err error
	return err
}
func (c *categoryproduct) delete(s *session) error {
	var err error
	return err
}

func (c *categoryproduct) selectProducts(s *session) ([]categoryproduct, error) {

	var categoryproducts []categoryproduct
	rows, err := s.selectCategoryProductByCategoryProductIDStmt.Query(c.ID)

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
			categoryproducts = append(categoryproducts, cp)
		}
	}

	err = rows.Err()
	return categoryproducts, err
}

// func (c *categoryproduct) indexesOf(slice []categoryinterface) []int {
// 	indexes := []int{}
// 	for i, ele := range slice {
// 		if strings.ToLower(ele.getName()) == strings.ToLower(c.getName()) {
// 			indexes = append(indexes, i)
// 		}
// 	}
// 	return indexes
// }

// func (c *categoryproduct) appendIfMissing(slice []categoryinterface) []categoryinterface {
// 	indexes := c.indexesOf(slice)
// 	if len(indexes) == 0 {
// 		return append(slice, c)
// 	}
// 	return slice
// }

// func (c *categoryproduct) removeIfPresent(slice []categoryinterface) []categoryinterface {
// 	indexes := c.indexesOf(slice)
// 	if len(indexes) > 0 {
// 		for i := range indexes {
// 			slice = append(slice[:i], slice[i+1:]...)
// 		}
// 	}
// 	return slice
// }

func syncProducts(s *session) error {
	var err error
	return err
}
