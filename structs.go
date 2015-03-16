package main

import (
	"encoding/json"
)

type entity interface {
	getDBAction() int
	insert(f *feed, s *session)
	update(f *feed, s *session)
	delete(f *feed, s *session)
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
