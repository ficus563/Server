package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

type Player struct {
	Name     string `json:"name"`
	HP       int    `json:"hp"`
	Strength int    `json:"strength"`
}

type Message struct {
	Author string `json:"author"`
	Text   string `json:"text"`
	Time   string `json:"time"`
}

type GameState struct {
	Players  map[string]Player `json:"players"`
	Messages []Message         `json:"messages"`
}

var (
	state = GameState{Players: make(map[string]Player), Messages: []Message{}}
	mu    sync.Mutex
)

func main() {
	http.HandleFunc("/sync", func(w http.ResponseWriter, r *http.Request) {
		var p Player
		if err := json.NewDecoder(r.Body).Decode(&p); err == nil {
			mu.Lock()
			if _, ok := state.Players[p.Name]; !ok { p.HP = 100 } // Респаун
			state.Players[p.Name] = p
			json.NewEncoder(w).Encode(state)
			mu.Unlock()
		}
	})

	http.HandleFunc("/chat", func(w http.ResponseWriter, r *http.Request) {
		var m Message
		if err := json.NewDecoder(r.Body).Decode(&m); err == nil {
			m.Time = time.Now().Format("15:04")
			mu.Lock()
			state.Messages = append(state.Messages, m)
			if len(state.Messages) > 15 { state.Messages = state.Messages[1:] }
			mu.Unlock()
		}
	})

	http.HandleFunc("/attack", func(w http.ResponseWriter, r *http.Request) {
		var req struct { Target string; Damage int; Attacker string }
		if err := json.NewDecoder(r.Body).Decode(&req); err == nil {
			mu.Lock()
			if t, ok := state.Players[req.Target]; ok {
				t.HP -= req.Damage
				if t.HP < 0 { t.HP = 0 }
				state.Players[req.Target] = t
				state.Messages = append(state.Messages, Message{
					Author: "БОЙ",
					Text:   fmt.Sprintf("%s жахнул %s на %d HP!", req.Attacker, req.Target, req.Damage),
					Time:   time.Now().Format("15:04"),
				})
			}
			mu.Unlock()
		}
	})

	fmt.Println("Сервер летит на :8080")
	http.ListenAndServe(":8080", nil)
}
