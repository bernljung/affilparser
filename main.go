package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"

	_ "github.com/go-sql-driver/mysql"
)

var addr = flag.String("addr", ":8001", "http service address")
var dbUser = flag.String("dbUser", "user", "database username")
var dbPassword = flag.String("dbPassword", "password", "database password")
var dbAddr = flag.String("dbAddr", "localhost", "database address")
var dbPort = flag.Int("dbPort", 3306, "database port")
var database = flag.String("database", "database", "database name")

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

		if len(s.feeds) > 0 {
			s.prepare()
			go s.update()
		}

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
