package main

import (
	"fmt"
	"io/ioutil"
	"strings"
	"net/url"
	"net/http"
	"./redis"
)

type Request struct {
	api string
	ops string
	course string
	primary_course string
	user string
	body []byte
	query map[string] []string
	cookies []*http.Cookie
	db redis.Client
}

type Status int

const (
	Success Status = iota
	Unauthorised
	BadRequest
	ServerError
)

type Response struct {
	code Status
	msg string
	cookie *http.Cookie
}

func MainHandler(w http.ResponseWriter, req *http.Request) {
	path := strings.Split(req.URL.Path, "/")[1:]
	if (path[0] == "api") {
		parsed_req := Request { cookies : req.Cookies() }
		switch len(path) {
		case 2:			// matches /api/:api
			parsed_req.api = path[1]
		case 3:			// matches /api/:api/:ops
			parsed_req.api = path[1]
			parsed_req.ops = path[2]
		case 4:			// matches /api/:api/:course/:ops
			parsed_req.api = path[1]
			parsed_req.ops = path[3]
			parsed_req.course = path[2]
		default:
			http.Error(w, "Invalid api request.", http.StatusBadRequest)
			return
		}
		query, err := url.ParseQuery(req.URL.RawQuery)
		if err != nil {
			http.Error(w, "Invalid query string.", http.StatusBadRequest)
			return
		}
		parsed_req.query = query
		body, err := ioutil.ReadAll(req.Body)
		if err != nil {
			http.Error(w, "Failed to read request body.", http.StatusInternalServerError)
			return
		}
		parsed_req.body = body
		res := HandleApi(parsed_req)
		switch res.code {
		case Success:
			if (res.cookie != nil) {
				http.SetCookie(w, res.cookie)
			}
			fmt.Fprintf(w, res.msg)
		case Unauthorised:
			http.Error(w, res.msg, http.StatusUnauthorized)
		case BadRequest:
			http.Error(w, res.msg, http.StatusBadRequest)
		case ServerError:
			http.Error(w, res.msg, http.StatusInternalServerError)
		}
	} else {
		http.ServeFile(w, req, "web" + req.URL.Path)
	}
}

func main() {
	http.HandleFunc("/", MainHandler)
	http.ListenAndServe(":3000", nil)
}
