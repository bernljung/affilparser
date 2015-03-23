package main

import (
	"encoding/json"
)

type entity interface {
	getDBAction() int
	getName() string
	getEntityType() string
	insert(s *session) error
	update(s *session) error
	delete(s *session) error
}

type message struct {
	feed   *feed
	entity entity
}

type Response struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

func (r Response) String() (s string) {
	b, err := json.Marshal(r)
	if err != nil {
		s = ""
		return
	}
	s = string(b)
	return
}
