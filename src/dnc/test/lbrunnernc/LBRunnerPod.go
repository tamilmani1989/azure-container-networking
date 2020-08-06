package main

import (
    "fmt"
    "log"
    "net/http"
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


func main() {
    http.HandleFunc("/hello", helloHandler)


    fmt.Printf("Starting server at port 9095\n")
    if err := http.ListenAndServe(":9095", nil); err != nil {
        log.Fatal(err)
    }
}