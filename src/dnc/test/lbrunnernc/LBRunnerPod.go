package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
)

const (
	getSourceIP = "https://api.ipify.org/?format=json"
)

func helloHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/hello" {
		http.Error(w, "404 not found.", http.StatusNotFound)
		return
	}

	if r.Method != "GET" {
		http.Error(w, "Method is not supported.", http.StatusNotFound)
		return
	}

	fmt.Fprintf(w, "LB Works!")
}

func lbMnatHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/lbmnat" {
		http.Error(w, "404 not found.", http.StatusNotFound)
		return
	}

	if r.Method != "GET" {
		http.Error(w, "Method is not supported.", http.StatusNotFound)
		return
	}

	resp, err := http.Get(getSourceIP)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		http.Error(w, "Return code was not OK", http.StatusInternalServerError)
		return
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if len(body) == 0 {
		http.Error(w, "Received bad output", http.StatusInternalServerError)
		return
	}

	fmt.Fprintf(w, string(body))
}

func main() {
	http.HandleFunc("/hello", helloHandler)
	http.HandleFunc("/lbmnat", lbMnatHandler)

	fmt.Printf("Starting server at port 9095\n")
	if err := http.ListenAndServe(":9095", nil); err != nil {
		log.Fatal(err)
	}
}
