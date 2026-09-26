package main

import (
	"encoding/json"
	"log"
	"net/http"

	"geoduel/internal/game"
)

func main() {
	lb := game.NewLeaderboard("leaderboard.json")
	hub := game.NewHub(lb)

	http.HandleFunc("/ws", hub.HandleWS)
	http.HandleFunc("/leaderboard", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(lb.Top(20))
	})
	noCache := func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Cache-Control", "no-store")
			h.ServeHTTP(w, r)
		})
	}
	http.Handle("/", noCache(http.FileServer(http.Dir("web"))))

	log.Println("listening on http://localhost:8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}
