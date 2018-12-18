package main

import (
	"fmt"

	"net/http"

        "github.com/gorilla/mux"

)

func handleWhoAreYou(w http.ResponseWriter, r *http.Request) {
        fmt.Fprintf(w, "Odin CMDB Server")
}

func handlePing(w http.ResponseWriter, r *http.Request) {
        fmt.Fprintf(w, "pong")
}

func handleDescription(w http.ResponseWriter, r *http.Request) {
        html := "Server Side Application For Odin CMDB"
        fmt.Fprintf(w, html)
}

func main() {

        router := mux.NewRouter()
        router.HandleFunc("/whoareyou", handleWhoAreYou)
        router.HandleFunc("/ping", handlePing)
        router.HandleFunc("/description", handleDescription)
        router.HandleFunc("/trigger", handleTrigger)
        router.HandleFunc("/jsonbodytrigger", handleJSONBodyTrigger)
        router.HandleFunc("/", handleHelp)

        err = http.ListenAndServe(":8088", router)
        if err != nil {
                fmt.Println("ListenAndServe: ", err)
        }
}
