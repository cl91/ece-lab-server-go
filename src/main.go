package main

import (
	"net/http"
)

func main() {
	http.HandleFunc("/", MainHandler)
	http.ListenAndServe(":4000", nil)
}
