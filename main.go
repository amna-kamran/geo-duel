package main

import (
	"log"
	"net/http"

	"geoduel/internal/game"
)

func main() {
	hub := game.NewHub()

	http.HandleFunc("/ws", hub.HandleWS)
	http.Handle("/", http.FileServer(http.Dir("web")))

	log.Println("listening on http://localhost:8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}
