package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

type Attack struct {
	From string; Zone int; Damage int
}

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
	pendingAttacks = make(map[string]Attack) // Кого бьют -> Информация об атаке
	mu    sync.Mutex
)

func main() {
	go func() {
		for {
			time.Sleep(2 * time.Second)
			mu.Lock()
			for name, p := range state.Players {
				if time.Since(p.LastSeen) > 10*time.Second { delete(state.Players, name) }
			}
			mu.Unlock()
		}
	}()

	http.HandleFunc("/sync", func(w http.ResponseWriter, r *http.Request) {
		var p Player
		json.NewDecoder(r.Body).Decode(&p)
		mu.Lock()
		defer mu.Unlock()
		
		if curr, ok := state.Players[p.Name]; !ok {
			p.HP, p.MaxHP, p.Str, p.Def, p.Coins = 100, 100, 15, 5, 100
			state.Players[p.Name] = p
		} else {
			curr.LastSeen = time.Now()
			state.Players[p.Name] = curr
		}
		json.NewEncoder(w).Encode(state)
	})

	http.HandleFunc("/action", func(w http.ResponseWriter, r *http.Request) {
		var req struct { Type, From, To string; Val, AtkZone, BlkZone int }
		json.NewDecoder(r.Body).Decode(&req)
		mu.Lock()
		defer mu.Unlock()

		p := state.Players[req.From]
		switch req.Type {
		case "chat": addMsg(req.From, req.To)
		case "pve":
			p.HP -= 10; p.Coins += 30
			if p.HP < 0 { p.HP = 0 }
			state.Players[req.From] = p
			// Проверка: не били ли нас, пока мы гуляли по лесу?
			checkIncoming(&p, req.BlkZone)
		case "pvp_init":
			pendingAttacks[req.To] = Attack{From: req.From, Zone: req.AtkZone, Damage: p.Str}
			addMsg("СИСТЕМА", fmt.Sprintf("%s замахнулся на %s!", req.From, req.To))
		case "shop":
			if req.To == "sword" && p.Coins >= 100 { p.Coins -= 100; p.Str += 20 }
			if req.To == "heal" && p.Coins >= 30 { p.Coins -= 30; p.HP = p.MaxHP }
			state.Players[req.From] = p
		}
	})
	http.ListenAndServe(":8080", nil)
}

func checkIncoming(p *Player, blkZone int) {
	if atk, ok := pendingAttacks[p.Name]; ok {
		if atk.Zone != blkZone {
			dmg := atk.Damage - p.Def
			if dmg < 10 { dmg = 10 }
			p.HP -= dmg
			addMsg("БОЙ", fmt.Sprintf("%s получил урон от %s (не угадал блок)!", p.Name, atk.From))
		} else {
			addMsg("БОЙ", fmt.Sprintf("%s заблокировал удар %s!", p.Name, atk.From))
		}
		delete(pendingAttacks, p.Name)
	}
}

func addMsg(a, t string) {
	state.Msgs = append(state.Msgs, struct{ A, T, Time string }{a, t, time.Now().Format("15:04")})
}
