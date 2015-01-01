package main

import (
	"net/http"
)

func main() {
	http.HandleFunc("/", MuxHandler)
	http.ListenAndServe(":4000", nil)
}
