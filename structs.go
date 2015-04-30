package main

import (
	"encoding/json"
)

type message struct {
	feed    *feed
	product product
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
