package game

import (
	"encoding/json"
	"log"
	"os"
	"sort"
	"sync"
)

type PlayerStats struct {
	Name       string `json:"name"`
	Games      int    `json:"games"`
	Wins       int    `json:"wins"`
	TotalScore int    `json:"totalScore"`
	BestScore  int    `json:"bestScore"`
}

func (s *PlayerStats) average() float64 {
	if s.Games == 0 {
		return 0
	}
	return float64(s.TotalScore) / float64(s.Games)
}

// Leaderboard tracks cumulative per-round stats across all games, keyed by
// player display name, and persists them to a JSON file so scores survive
// server restarts.
type Leaderboard struct {
	mu    sync.Mutex
	path  string
	stats map[string]*PlayerStats
}

func NewLeaderboard(path string) *Leaderboard {
	lb := &Leaderboard{path: path, stats: make(map[string]*PlayerStats)}
	lb.load()
	return lb
}

func (lb *Leaderboard) load() {
	data, err := os.ReadFile(lb.path)
	if err != nil {
		return // no file yet; start fresh
	}
	var list []PlayerStats
	if err := json.Unmarshal(data, &list); err != nil {
		log.Println("leaderboard: failed to parse", lb.path, err)
		return
	}
	for i := range list {
		s := list[i]
		lb.stats[s.Name] = &s
	}
}

// save must be called with lb.mu held.
func (lb *Leaderboard) save() {
	list := make([]PlayerStats, 0, len(lb.stats))
	for _, s := range lb.stats {
		list = append(list, *s)
	}
	data, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		log.Println("leaderboard: marshal error:", err)
		return
	}
	if err := os.WriteFile(lb.path, data, 0644); err != nil {
		log.Println("leaderboard: write error:", err)
	}
}

// Record adds one round's result for a real (non-bot) player.
func (lb *Leaderboard) Record(name string, score int, won bool) {
	if name == "" {
		return
	}
	lb.mu.Lock()
	defer lb.mu.Unlock()

	s, ok := lb.stats[name]
	if !ok {
		s = &PlayerStats{Name: name}
		lb.stats[name] = s
	}
	s.Games++
	s.TotalScore += score
	if score > s.BestScore {
		s.BestScore = score
	}
	if won {
		s.Wins++
	}
	lb.save()
}

// AverageFor returns the player's average round score, or a neutral
// default for players with no history yet.
func (lb *Leaderboard) AverageFor(name string) float64 {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	s, ok := lb.stats[name]
	if !ok || s.Games == 0 {
		return 2200 // roughly a "decent, not great" guess for a new player
	}
	return s.average()
}

// Top returns up to n players sorted by total score, descending.
func (lb *Leaderboard) Top(n int) []PlayerStats {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	list := make([]PlayerStats, 0, len(lb.stats))
	for _, s := range lb.stats {
		list = append(list, *s)
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].TotalScore > list[j].TotalScore
	})
	if len(list) > n {
		list = list[:n]
	}
	return list
}
