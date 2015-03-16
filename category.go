package main

import (
	"fmt"
	"log"
)

type category struct {
	ID                int
	CategoryProductID int
	Name              string
	DBAction          int
}

func (c category) getDBAction() int {
	return c.DBAction
}

func (c category) insert(f *feed, s *session) {
	_, err := s.db.Exec(
		"INSERT INTO categories (name, created_at, updated_at) "+
			"VALUES (?,now(),now())",
		c.Name,
	)

	if err != nil {
		f.DBOperationError <- err
	} else {
		f.DBOperationDone <- fmt.Sprintf("New category: '%v'.", c.Name)
	}
}

func (c category) update(f *feed, s *session) {
	f.DBOperationDone <- fmt.Sprintf("Updated category: '%v'.", c.Name)
}

func (c category) delete(f *feed, s *session) {
	f.DBOperationDone <- fmt.Sprintf("No delete action, category: '%v'.",
		c.Name)
}

func (c category) indexesOf(slice []category) []int {
	indexes := []int{}
	for i, ele := range slice {
		if ele.Name == c.Name {
			indexes = append(indexes, i)
		}
	}
	return indexes
}

func (c category) appendIfMissing(slice []category) []category {
	indexes := c.indexesOf(slice)
	if len(indexes) == 0 {
		return append(slice, c)
	}
	return slice
}

func (c category) removeIfPresent(slice []category) []category {
	indexes := c.indexesOf(slice)
	log.Println(indexes, c.Name)
	if len(indexes) > 0 {
		for i := range indexes {
			slice = append(slice[:i], slice[i+1:]...)
		}
	}
	return slice
}
