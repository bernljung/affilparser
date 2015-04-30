package main

type categoryinterface interface {
	// indexesOf([]categoryinterface) []int
	getName() string
	getCategoryID() int
	setCategoryID(int)
	setSiteID(int)
	getID() int
	getSearchString() string
	getEntityType() string
	// appendIfMissing([]categoryinterface) []categoryinterface
	// removeIfPresent([]categoryinterface) []categoryinterface
	getDBAction() int
	setDBAction(int)
	getCreatedByID() int
	setCreatedByID(int)
	getDescription() string
	selectProducts(s *session) ([]categoryproduct, error)
	insert(s *session) error
	update(s *session) error
	delete(s *session) error
	syncProducts(s *session) error
}
