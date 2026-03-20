package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

type Player struct {
	Name string `json:"name"`
	HP   int    `json:"hp"`
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
	state = GameState{
		Players:  make(map[string]Player),
		Messages: []Message{},
	}
	mu sync.Mutex
)

func main() {
	// Синхронизация данных
	http.HandleFunc("/sync", func(w http.ResponseWriter, r *http.Request) {
		var p Player
		if err := json.NewDecoder(r.Body).Decode(&p); err == nil {
			mu.Lock()
			state.Players[p.Name] = p
			json.NewEncoder(w).Encode(state)
			mu.Unlock()
		}
	})

	// Отправка сообщений и PvP команд
	http.HandleFunc("/chat", func(w http.ResponseWriter, r *http.Request) {
		var msg Message
		json.NewDecoder(r.Body).Decode(&msg)
		msg.Time = time.Now().Format("15:04")
		mu.Lock()
		state.Messages = append(state.Messages, msg)
		if len(state.Messages) > 10 { state.Messages = state.Messages[1:] }
		mu.Unlock()
	})

	// Обработка атаки (PvP)
	http.HandleFunc("/attack", func(w http.ResponseWriter, r *http.Request) {
		var req struct { Target string; Damage int; Attacker string }
		if err := json.NewDecoder(r.Body).Decode(&req); err == nil {
			mu.Lock()
			if target, ok := state.Players[req.Target]; ok {
				target.HP -= req.Damage
				if target.HP < 0 { target.HP = 0 }
				state.Players[req.Target] = target
				// Системное сообщение в чат
				state.Messages = append(state.Messages, Message{
					Author: "БОЙ",
					Text:   fmt.Sprintf("%s ударил %s на %d!", req.Attacker, req.Target, req.Damage),
					Time:   time.Now().Format("15:04"),
				})
			}
			mu.Unlock()
		}
	})

	fmt.Println("Сервер запущен на :8080")
	http.ListenAndServe(":8080", nil)
}
