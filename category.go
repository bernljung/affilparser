package main

type categoryinterface interface {
	indexesOf([]categoryinterface) []int
	getName() string
	getCategoryID() int
	setCategoryID(int)
	setSiteID(int)
	getID() int
	getKeywords() string
	getEntityType() string
	appendIfMissing([]categoryinterface) []categoryinterface
	removeIfPresent([]categoryinterface) []categoryinterface
	getDBAction() int
	setDBAction(int)
	getCreatedByID() int
	setCreatedByID(int)
	getDescriptionByUser() string
	insert(s *session) error
	update(s *session) error
	delete(s *session) error
}

type category struct {
	ID                int
	ParentID          int
	Name              string
	Slug              string
	SiteID            int
	Keywords          string
	DescriptionByUser string
	CreatedByID       int
	DBAction          int
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

func (c *category) getDescriptionByUser() string {
	return c.DescriptionByUser
}

func (c *category) getKeywords() string {
	return c.Keywords
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

func (c *category) indexesOf(slice []categoryinterface) []int {
	indexes := []int{}
	for i, ele := range slice {
		if ele.getName() == c.getName() {
			indexes = append(indexes, i)
		}
	}
	return indexes
}

func (c *category) insert(s *session) error {
	_, err := s.db.Exec(
		"INSERT INTO categories (name, slug, site_id, created_by_id, created_at, updated_at) "+
			"VALUES (?,?,?,?,now(),now())",
		c.Name,
		c.Slug,
		c.SiteID,
		CREATED_BY_FEED,
	)
	return err
}

func (c *category) update(s *session) error {
	var err error
	return err
}

func (c *category) delete(s *session) error {
	_, err := s.db.Exec("DELETE FROM categories WHERE id = ?", c.ID)
	return err
}

func (c *category) appendIfMissing(slice []categoryinterface) []categoryinterface {
	indexes := c.indexesOf(slice)
	if len(indexes) == 0 {
		return append(slice, c)
	}
	return slice
}

func (c *category) removeIfPresent(slice []categoryinterface) []categoryinterface {
	indexes := c.indexesOf(slice)
	if len(indexes) > 0 {
		for i := len(indexes) - 1; i >= 0; i-- {
			slice = append(slice[:i], slice[i+1:]...)
		}
	}
	return slice
}

type categoryproduct struct {
	category
	ID          int
	CategoryID  int
	ProductID   int
	CreatedByID int
}

func (c *categoryproduct) getName() string {
	return c.Name
}

func (c *categoryproduct) setSiteID(siteID int) {

}

func (c *categoryproduct) getKeywords() string {
	return c.Keywords
}

func (c *categoryproduct) getDescriptionByUser() string {
	return c.DescriptionByUser
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
	c.DBAction = action
}

func (c *categoryproduct) getID() int {
	return c.ID
}

func (c *categoryproduct) setCategoryID(id int) {
	c.CategoryID = id
}

func (c *categoryproduct) indexesOf(slice []categoryinterface) []int {
	indexes := []int{}
	for i, ele := range slice {
		if ele.getName() == c.getName() {
			indexes = append(indexes, i)
		}
	}
	return indexes
}

func (c *categoryproduct) appendIfMissing(slice []categoryinterface) []categoryinterface {
	indexes := c.indexesOf(slice)
	if len(indexes) == 0 {
		return append(slice, c)
	}
	return slice
}

func (c *categoryproduct) removeIfPresent(slice []categoryinterface) []categoryinterface {
	indexes := c.indexesOf(slice)
	if len(indexes) > 0 {
		for i := range indexes {
			slice = append(slice[:i], slice[i+1:]...)
		}
	}
	return slice
}
