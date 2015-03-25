package main

import "encoding/json"

type site struct {
	ID        int64
	Name      string
	Subdomain string
}

func (site site) String() (s string) {
	b, err := json.Marshal(site)
	if err != nil {
		s = ""
		return
	}
	s = string(b)
	return
}
