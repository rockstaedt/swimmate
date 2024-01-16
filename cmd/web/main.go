package main

import (
	"log"
	"net/http"
)

func main() {
	mux := http.NewServeMux()

	fileServer := http.FileServer(http.Dir("./ui/static/"))
	mux.Handle("/static/", http.StripPrefix("/static", fileServer))

	mux.HandleFunc("/", home)

	log.Print("starting server on :8998")

	err := http.ListenAndServe(":8998", mux)
	log.Fatal(err)
}
