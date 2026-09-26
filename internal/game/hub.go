package game

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// botWaitTimeout is how long a solo player waits for a real opponent
// before getting paired against GeoBot instead.
const botWaitTimeout = 8 * time.Second

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

type LatLng struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
}

type clientMsg struct {
	Type string  `json:"type"`
	Lat  float64 `json:"lat,omitempty"`
	Lng  float64 `json:"lng,omitempty"`
}

type serverMsg struct {
	Type          string  `json:"type"`
	Place         string  `json:"place,omitempty"`
	Continent     string  `json:"continent,omitempty"`
	OpponentName  string  `json:"opponentName,omitempty"`
	IsBot         bool    `json:"isBot,omitempty"`
	Actual        *LatLng `json:"actual,omitempty"`
	YourGuess     *LatLng `json:"yourGuess,omitempty"`
	OpponentGuess *LatLng `json:"opponentGuess,omitempty"`
	YourScore     int     `json:"yourScore,omitempty"`
	OpponentScore int     `json:"opponentScore,omitempty"`
	YourTotal     int     `json:"yourTotal,omitempty"`
	OpponentTotal int     `json:"opponentTotal,omitempty"`
}

// Player wraps one WebSocket connection. Each Player runs its own
// read and write goroutines, which is the idiomatic Gorilla pattern.
type Player struct {
	conn   *websocket.Conn
	name   string
	send   chan []byte
	paired chan *Game
	game   *Game
	idx    int // 0 or 1 within its Game
}

func newPlayer(conn *websocket.Conn, name string) *Player {
	return &Player{
		conn:   conn,
		name:   name,
		send:   make(chan []byte, 8),
		paired: make(chan *Game, 1),
	}
}

func sanitizeName(raw string) string {
	name := strings.TrimSpace(raw)
	if len(name) > 24 {
		name = name[:24]
	}
	if name == "" {
		name = fmt.Sprintf("Guest%d", rand.Intn(9000)+1000)
	}
	return name
}

func (p *Player) sendJSON(v any) {
	data, err := json.Marshal(v)
	if err != nil {
		log.Println("marshal error:", err)
		return
	}
	select {
	case p.send <- data:
	default: // buffer full or closed; drop rather than block
	}
}

func (p *Player) writePump() {
	for msg := range p.send {
		if err := p.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
			return
		}
	}
}

func (p *Player) readPump() {
	defer func() {
		p.conn.Close()
		close(p.send)
	}()
	for {
		_, data, err := p.conn.ReadMessage()
		if err != nil {
			return
		}
		var msg clientMsg
		if err := json.Unmarshal(data, &msg); err != nil {
			continue
		}
		if p.game == nil {
			continue
		}
		switch msg.Type {
		case "guess":
			p.game.submitGuess(p.idx, LatLng{Lat: msg.Lat, Lng: msg.Lng})
		case "ready":
			p.game.submitReady(p.idx)
		}
	}
}

// Game coordinates one round between two players. players[1] is nil when
// vsBot is true, in which case GeoBot plays that seat automatically. All
// state mutation goes through mu, since both players' goroutines (plus the
// bot's own goroutine) touch it.
type Game struct {
	mu       sync.Mutex
	players  [2]*Player
	lb       *Leaderboard
	vsBot    bool
	location Location
	guesses  [2]*LatLng
	ready    [2]bool
	scores   [2]int
}

func newGame(a, b *Player, lb *Leaderboard) *Game {
	return &Game{players: [2]*Player{a, b}, lb: lb, vsBot: b == nil}
}

func (g *Game) opponentName(forIdx int) string {
	other := g.players[1-forIdx]
	if other != nil {
		return other.name
	}
	return botName
}

func (g *Game) startRound() {
	g.mu.Lock()
	g.location = RandomLocation()
	g.guesses = [2]*LatLng{nil, nil}
	loc := g.location
	place := loc.Name
	g.mu.Unlock()

	for i, p := range g.players {
		if p == nil {
			continue
		}
		p.sendJSON(serverMsg{
			Type:         "round_start",
			Place:        place,
			OpponentName: g.opponentName(i),
			IsBot:        g.vsBot,
		})
	}

	if g.vsBot {
		go g.botPlay(loc)
	}
}

func (g *Game) botPlay(loc Location) {
	delay := time.Duration(1500+rand.Intn(3000)) * time.Millisecond
	time.Sleep(delay)

	avg := 2200.0
	if g.lb != nil && g.players[0] != nil {
		avg = g.lb.AverageFor(g.players[0].name)
	}
	g.submitGuess(1, botGuess(loc, avg))
}

func (g *Game) submitGuess(idx int, guess LatLng) {
	g.mu.Lock()
	if g.guesses[idx] != nil {
		g.mu.Unlock()
		return // already guessed this round
	}
	g.guesses[idx] = &guess
	bothIn := g.guesses[0] != nil && g.guesses[1] != nil
	g.mu.Unlock()

	if bothIn {
		g.finishRound()
	}
}

func (g *Game) finishRound() {
	g.mu.Lock()
	loc := g.location
	guesses := g.guesses
	d0 := HaversineKm(guesses[0].Lat, guesses[0].Lng, loc.Lat, loc.Lng)
	d1 := HaversineKm(guesses[1].Lat, guesses[1].Lng, loc.Lat, loc.Lng)
	s0, s1 := Score(d0), Score(d1)
	g.scores[0] += s0
	g.scores[1] += s1
	scores := g.scores
	g.ready = [2]bool{false, false}
	g.mu.Unlock()

	if g.lb != nil {
		if p := g.players[0]; p != nil {
			g.lb.Record(p.name, s0, s0 > s1)
		}
		if p := g.players[1]; p != nil {
			g.lb.Record(p.name, s1, s1 > s0)
		}
	}

	actual := &LatLng{Lat: loc.Lat, Lng: loc.Lng}
	roundScores := [2]int{s0, s1}

	for i, p := range g.players {
		if p == nil {
			continue
		}
		you, opp := i, 1-i
		p.sendJSON(serverMsg{
			Type:          "round_result",
			Continent:     loc.Continent,
			Actual:        actual,
			YourGuess:     guesses[you],
			OpponentGuess: guesses[opp],
			YourScore:     roundScores[you],
			OpponentScore: roundScores[opp],
			YourTotal:     scores[you],
			OpponentTotal: scores[opp],
		})
	}
}

func (g *Game) submitReady(idx int) {
	g.mu.Lock()
	g.ready[idx] = true
	bothReady := g.ready[0] && (g.ready[1] || g.vsBot)
	g.mu.Unlock()

	if bothReady {
		g.startRound()
	}
}

// Hub pairs up incoming connections two at a time. A player who waits
// longer than botWaitTimeout without a human opponent is paired with
// GeoBot instead.
type Hub struct {
	mu      sync.Mutex
	waiting *Player
	lb      *Leaderboard
}

func NewHub(lb *Leaderboard) *Hub {
	return &Hub{lb: lb}
}

func (h *Hub) fallBackToBot(p *Player) {
	time.Sleep(botWaitTimeout)

	h.mu.Lock()
	if h.waiting != p {
		h.mu.Unlock()
		return // already paired with a human (or replaced) in the meantime
	}
	h.waiting = nil
	h.mu.Unlock()

	g := newGame(p, nil, h.lb)
	select {
	case p.paired <- g:
		g.startRound()
	default:
	}
}

func (h *Hub) HandleWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("upgrade error:", err)
		return
	}

	name := sanitizeName(r.URL.Query().Get("name"))
	p := newPlayer(conn, name)
	go p.writePump()

	h.mu.Lock()
	if h.waiting == nil {
		h.waiting = p
		h.mu.Unlock()
		p.sendJSON(serverMsg{Type: "waiting"})
		go h.fallBackToBot(p)
		p.game = <-p.paired // blocks until a human or GeoBot joins
		p.idx = 0
	} else {
		opponent := h.waiting
		h.waiting = nil
		h.mu.Unlock()

		g := newGame(opponent, p, h.lb)
		p.game = g
		p.idx = 1
		opponent.paired <- g // wakes opponent's goroutine
		g.startRound()
	}

	p.readPump()

	h.mu.Lock()
	if h.waiting == p {
		h.waiting = nil
	}
	h.mu.Unlock()
}
