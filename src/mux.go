package main

import (
	"fmt"
	"strings"
	"net/url"
	"net/http"
)

type Request struct {
	api string
	ops string
	param string
	query map[string] []string
}

type Status int

const (
	Success Status = 1 + iota
	Unauthorised
	BadRequest
	ServerError
)

type Response struct {
	code Status
	msg string
}

func MuxHandler(w http.ResponseWriter, req *http.Request) {
	path := strings.Split(req.URL.Path, "/")[1:]
	if (path[0] == "api") {
		var api string
		var ops string
		var param string
		switch len(path) {
		case 2:			// matches /api/:api
			// do nothing
		case 3:			// matches /api/:api/:ops
			ops = path[3]
		case 4:			// matches /api/:api/:param/:ops
			ops = path[3]
			param = path[4]
		default:
			http.Error(w, "Invalid api request.", http.StatusBadRequest)
			return
		}
		query, err := url.ParseQuery(req.URL.RawQuery)
		if err != nil {
			http.Error(w, "Invalid query string.", http.StatusBadRequest)
			return
		}
		parsed_req := Request {	api, ops, param, query }
		res := HandleApi(parsed_req)
		switch res.code {
		case Success:
			fmt.Fprintf(w, res.msg)
		case Unauthorised:
			fmt.Fprintf(w, res.msg, http.StatusUnauthorized)
		case BadRequest:
			http.Error(w, res.msg, http.StatusBadRequest)
		case ServerError:
			http.Error(w, res.msg, http.StatusInternalServerError)
		}
	} else {
		http.ServeFile(w, req, "web" + req.URL.Path)
	}
}
