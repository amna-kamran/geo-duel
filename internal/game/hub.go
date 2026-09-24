package game

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

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
	send   chan []byte
	paired chan *Game
	game   *Game
	idx    int // 0 or 1 within its Game
}

func newPlayer(conn *websocket.Conn) *Player {
	return &Player{
		conn:   conn,
		send:   make(chan []byte, 8),
		paired: make(chan *Game, 1),
	}
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
		switch msg.Type {
		case "guess":
			p.game.submitGuess(p.idx, LatLng{Lat: msg.Lat, Lng: msg.Lng})
		case "ready":
			p.game.submitReady(p.idx)
		}
	}
}

// Game coordinates one round between two paired players. All state
// mutation goes through mu, since both players' goroutines touch it.
type Game struct {
	mu       sync.Mutex
	players  [2]*Player
	location Location
	guesses  [2]*LatLng
	ready    [2]bool
	scores   [2]int
}

func newGame(a, b *Player) *Game {
	return &Game{players: [2]*Player{a, b}}
}

func (g *Game) startRound() {
	g.mu.Lock()
	g.location = RandomLocation()
	g.guesses = [2]*LatLng{nil, nil}
	place := g.location.Name
	g.mu.Unlock()

	for _, p := range g.players {
		p.sendJSON(serverMsg{Type: "round_start", Place: place})
	}
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

	actual := &LatLng{Lat: loc.Lat, Lng: loc.Lng}
	roundScores := [2]int{s0, s1}

	for i, p := range g.players {
		you, opp := i, 1-i
		p.sendJSON(serverMsg{
			Type:          "round_result",
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
	bothReady := g.ready[0] && g.ready[1]
	g.mu.Unlock()

	if bothReady {
		g.startRound()
	}
}

// Hub pairs up incoming connections two at a time.
type Hub struct {
	mu      sync.Mutex
	waiting *Player
}

func NewHub() *Hub {
	return &Hub{}
}

func (h *Hub) HandleWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("upgrade error:", err)
		return
	}

	p := newPlayer(conn)
	go p.writePump()

	h.mu.Lock()
	if h.waiting == nil {
		h.waiting = p
		h.mu.Unlock()
		p.sendJSON(serverMsg{Type: "waiting"})
		p.game = <-p.paired // blocks until a second player joins
		p.idx = 0
	} else {
		opponent := h.waiting
		h.waiting = nil
		h.mu.Unlock()

		g := newGame(opponent, p)
		p.game = g
		p.idx = 1
		opponent.paired <- g // wakes opponent's goroutine
		g.startRound()
	}

	p.readPump()
}
