package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

type Message struct {
	Author string `json:"author"`
	Text   string `json:"text"`
	Time   string `json:"time"`
}

type Player struct {
	Name     string `json:"name"`
	HP       int    `json:"hp"`
	MaxHP    int    `json:"max_hp"`
	Level    int    `json:"level"`
	Strength int    `json:"strength"`
	LastSeen time.Time `json:"-"` // Для очистки вышедших игроков
}

type GameState struct {
	Players  map[string]*Player `json:"players"`
	Messages []Message          `json:"messages"`
}

var (
	state = GameState{
		Players:  make(map[string]*Player),
     Messages: []Message{{Author: "Система", Text: "Добро пожаловать в RPG!", Time: ""}},
	}
	mu sync.Mutex
)

func main() {
	// Основной эндпоинт для синхронизации
	http.HandleFunc("/sync", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()

		if r.Method == http.MethodPost {
			var p Player
			if err := json.NewDecoder(r.Body).Decode(&p); err == nil {
				p.LastSeen = time.Now()
				state.Players[p.Name] = &p
			}
		}
		json.NewEncoder(w).Encode(state)
	})

	// Эндпоинт для отправки сообщений в чат
	http.HandleFunc("/chat", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			var msg Message
			if err := json.NewDecoder(r.Body).Decode(&msg); err == nil {
				mu.Lock()
				msg.Time = time.Now().Format("15:04")
				state.Messages = append(state.Messages, msg)
				if len(state.Messages) > 20 { // Храним последние 20 сообщений
					state.Messages = state.Messages[1:]
				}
				mu.Unlock()
			}
		}
	})

	fmt.Println("Сервер запущен на :8080")
	http.ListenAndServe(":8080", nil)
}
