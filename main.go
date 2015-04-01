package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"

	_ "github.com/go-sql-driver/mysql"
)

var addr = flag.String("addr", ":8080", "http service address")
var DSN = flag.String("dsn", ":8080", "username:password@tcp(address:port)/database")

func getSession(req *http.Request) (session, Response) {
	var s session
	var resp Response
	site := req.FormValue("site")
	err := s.init(site)
	if err != nil {
		resp = Response{Success: false, Message: err.Error()}
	} else {
		err := s.selectFeeds()
		if err != nil {
			resp = Response{Success: false, Message: err.Error()}
		} else {
			if len(s.feeds) < 1 {
				resp = Response{Success: false, Message: "No feeds to parse."}
			} else {
				resp = Response{Success: true, Message: "Starting."}
			}
		}
	}
	return s, resp
}

// handler handles incoming requests for feed updates.
// the feed is validated and passed on to f.fetch chan.
func updateFeedsHandler(rw http.ResponseWriter, req *http.Request) {
	if req.Method == "POST" {
		rw.Header().Set("Content-Type", "application/json")

		s, resp := getSession(req)

		fmt.Fprint(rw, resp)
		s.prepare()
		go s.update()

	} else {
		http.NotFound(rw, req)
	}
}

func refreshHandler(rw http.ResponseWriter, req *http.Request) {
	if req.Method == "POST" {
		rw.Header().Set("Content-Type", "application/json")
		s, resp := getSession(req)

		fmt.Fprint(rw, resp)
		s.prepare()
		go s.refresh()
	} else {
		http.NotFound(rw, req)
	}
}

func main() {
	flag.Parse()
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	http.HandleFunc("/updatefeeds", updateFeedsHandler)
	http.HandleFunc("/refresh", refreshHandler)

	message := fmt.Sprintf("Starting server on %v", *addr)
	log.Println(message)
	if err := http.ListenAndServe(*addr, nil); err != nil {
		log.Fatal("ListenAndServe:", err)
	}
}
