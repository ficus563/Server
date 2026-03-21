package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

type Player struct {
	Name string `json:"name"`; HP int `json:"hp"`; MaxHP int `json:"mhp"`
	Str int `json:"str"`; Def int `json:"def"`; Coins int `json:"coins"`
	LastSeen time.Time `json:"-"`
}

type GameState struct {
	Players map[string]Player `json:"players"`
	Msgs    []struct{ A, T, Time string } `json:"msgs"`
}

var (
	state = GameState{Players: make(map[string]Player)}
	mu    sync.Mutex
)

func main() {
	// Очистка мертвых душ (кто вышел)
	go func() {
		for {
			time.Sleep(2 * time.Second)
			mu.Lock()
			for name, p := range state.Players {
				if time.Since(p.LastSeen) > 5*time.Second {
					delete(state.Players, name)
				}
			}
			mu.Unlock()
		}
	}()

	http.HandleFunc("/sync", func(w http.ResponseWriter, r *http.Request) {
		var p Player
		if err := json.NewDecoder(r.Body).Decode(&p); err == nil {
			mu.Lock()
			curr, exists := state.Players[p.Name]
			if !exists {
				p.HP, p.MaxHP, p.Str, p.Def, p.Coins = 100, 100, 15, 5, 100
				state.Players[p.Name] = p
			} else {
				curr.LastSeen = time.Now()
				state.Players[p.Name] = curr
			}
			json.NewEncoder(w).Encode(state)
			mu.Unlock()
		}
	})

	http.HandleFunc("/action", func(w http.ResponseWriter, r *http.Request) {
		var req struct { Type, From, To string; Val int; AtkZone, BlkZone int }
		json.NewDecoder(r.Body).Decode(&req)
		mu.Lock()
		defer mu.Unlock()

		p := state.Players[req.From]
		switch req.Type {
		case "chat": addMsg(req.From, req.To)
		case "pve":
			p.HP -= req.Val
			p.Coins += 40
			if p.HP < 0 { p.HP = 0 }
			state.Players[req.From] = p
		case "pvp":
			if t, ok := state.Players[req.To]; ok {
				dmg := 0
				if req.AtkZone != req.BlkZone { // Если не заблочил
					dmg = p.Str - t.Def
					if dmg < 10 { dmg = 10 }
					t.HP -= dmg
					if t.HP < 0 { t.HP = 0 }
					state.Players[req.To] = t
					addMsg("БОЙ", fmt.Sprintf("%s пробил %s! -%d HP", req.From, req.To, dmg))
				} else {
					addMsg("БОЙ", fmt.Sprintf("%s заблокировал удар %s!", req.To, req.From))
				}
			}
		case "shop":
			switch req.To {
			case "sword": if p.Coins >= 100 { p.Coins -= 100; p.Str += 20 }
			case "armor": if p.Coins >= 80 { p.Coins -= 80; p.Def += 15 }
			case "heal":  if p.Coins >= 30 { p.Coins -= 30; p.HP = p.MaxHP }
			}
			state.Players[req.From] = p
		}
	})
	fmt.Println("СЕРВЕР ОБНОВЛЕН (Port 8080)"); http.ListenAndServe(":8080", nil)
}

func addMsg(a, t string) {
	state.Msgs = append(state.Msgs, struct{ A, T, Time string }{a, t, time.Now().Format("15:04")})
}
