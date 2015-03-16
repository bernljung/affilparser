package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"

	_ "github.com/go-sql-driver/mysql"
)

var addr = flag.String("addr", ":8080", "http service address")
var DSN = "homestead:secret@tcp(localhost:33060)/"

// handler handles incoming requests for feed updates.
// the feed is validated and passed on to f.fetch chan.
func handler(rw http.ResponseWriter, req *http.Request) {
	if req.Method == "POST" {
		rw.Header().Set("Content-Type", "application/json")

		var s session
		dbString := req.FormValue("database")
		err := s.init(dbString)

		if err != nil {
			fmt.Fprint(rw, Response{
				Success: false,
				Message: err.Error(),
			})
		} else {
			err := s.getFeeds()
			if err != nil {
				fmt.Fprint(rw, Response{
					Success: false,
					Message: err.Error(),
				})
			} else {
				if len(s.feeds) < 1 {
					fmt.Fprint(rw, Response{
						Success: false,
						Message: "No feeds to parse.",
					})
				} else {
					fmt.Fprint(rw, Response{
						Success: true,
						Message: "Starting.",
					})
					go s.run()
				}
			}
		}
	} else {
		http.NotFound(rw, req)
	}
}

func main() {
	flag.Parse()

	http.HandleFunc("/go", handler)

	message := fmt.Sprintf("Starting server on %v", *addr)
	log.Println(message)
	if err := http.ListenAndServe(*addr, nil); err != nil {
		log.Fatal("ListenAndServe:", err)
	}
}
