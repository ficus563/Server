package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type Character struct {
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
	Players  map[string]Character `json:"players"`
	Messages []Message            `json:"messages"`
}

var (
	hero        Character
	app         *tview.Application
	pages       *tview.Pages
	chatView    *tview.TextView
	playersList *tview.TextView
	serverURL   = "http://localhost:8080" 
)

func networkLoop() {
	for {
		data, _ := json.Marshal(hero)
		resp, err := http.Post(serverURL+"/sync", "application/json", bytes.NewBuffer(data))
		if err == nil {
			var gs GameState
			if err := json.NewDecoder(resp.Body).Decode(&gs); err == nil {
				app.QueueUpdateDraw(func() {
					playersList.Clear()
					fmt.Fprintln(playersList, "[yellow]ИГРОКИ:[-]")
					for name, p := range gs.Players {
						color := "white"
						if name == hero.Name { 
							color = "green"
							hero.HP = p.HP 
						}
						fmt.Fprintf(playersList, "[%s]• %s (HP: %d)[-]\n", color, name, p.HP)
					}
					chatView.Clear()
					for _, m := range gs.Messages {
						fmt.Fprintf(chatView, "[gray]%s[-] [blue]%s:[-] %s\n", m.Time, m.Author, m.Text)
					}
					chatView.ScrollToEnd()
				})
			}
			resp.Body.Close()
		}
		time.Sleep(1 * time.Second)
	}
}

func pvpAttack(target string) {
	if target == "" || target == hero.Name { return }
	dmg := hero.Strength + rand.Intn(5)
	payload, _ := json.Marshal(map[string]interface{}{
		"Target": target, "Damage": dmg, "Attacker": hero.Name,
	})
	go http.Post(serverURL+"/attack", "application/json", bytes.NewBuffer(payload))
}

func showMenu() {
	menuList := tview.NewList().
		AddItem("В лес (PvE)", "Бой с мобом", '1', func() { startBattle() }).
		AddItem("АТАКОВАТЬ ИГРОКА", "Введите ник для PvP", '2', func() { showPvPDialog() }).
		AddItem("Выход", "", 'q', func() { app.Stop() })
	menuList.SetBorder(true).SetTitle(" МЕНЮ ")

	chatView = tview.NewTextView().SetDynamicColors(true).SetWordWrap(true)
	chatView.SetBorder(true).SetTitle(" ЧАТ ")

	inputField := tview.NewInputField().SetLabel("> ")
	inputField.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEnter {
			msg, _ := json.Marshal(Message{Author: hero.Name, Text: inputField.GetText()})
			go http.Post(serverURL+"/chat", "application/json", bytes.NewBuffer(msg))
			inputField.SetText("")
		}
	})
	inputField.SetBorder(true).SetTitle(" СООБЩЕНИЕ ")

	playersList = tview.NewTextView().SetDynamicColors(true)
	playersList.SetBorder(true).SetTitle(" ОНЛАЙН ")

	mainLayout := tview.NewFlex().
		AddItem(menuList, 20, 1, true).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(chatView, 0, 1, false).
			AddItem(inputField, 3, 1, true), 0, 2, false).
		AddItem(playersList, 20, 1, false)

	pages.AddPage("menu", mainLayout, true, true)
	go networkLoop()
}

func showPvPDialog() {
	form := tview.NewForm().
		AddInputField("Ник цели", "", 20, nil, nil)
	form.AddButton("УДАРИТЬ!", func() {
		target := form.GetFormItem(0).(*tview.InputField).GetText()
		pvpAttack(target)
		pages.SwitchToPage("menu")
	}).AddButton("Отмена", func() { pages.SwitchToPage("menu") })
	
	form.SetBorder(true).SetTitle(" PvP АТАКА ")
	pages.AddPage("pvp", form, true, true)
	pages.SwitchToPage("pvp")
}

func startBattle() {
	mHP := 50
	battleInfo := tview.NewTextView().SetDynamicColors(true)
	battleList := tview.NewList().
		AddItem("УДАР", "", 'a', func() {
			mHP -= hero.Strength
			hero.HP -= 5
			battleInfo.SetText(fmt.Sprintf("[red]Монстр HP: %d[-]\n[green]Ваше HP: %d[-]", mHP, hero.HP))
			if mHP <= 0 || hero.HP <= 0 { pages.SwitchToPage("menu") }
		}).
		AddItem("Сбежать", "", 'e', func() { pages.SwitchToPage("menu") })
	
	layout := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(battleInfo, 0, 1, false).
		AddItem(battleList, 10, 1, true)
	pages.AddPage("battle", layout, true, true)
	pages.SwitchToPage("battle")
}

func main() {
	rand.Seed(time.Now().UnixNano())
	app = tview.NewApplication()
	pages = tview.NewPages()
	
	f := tview.NewForm().AddInputField("Ник", "ficus", 20, nil, nil)
	f.AddButton("СТАРТ", func() {
		name := f.GetFormItem(0).(*tview.InputField).GetText()
		hero = Character{Name: name, HP: 100, Strength: 10}
		showMenu()
		pages.SwitchToPage("menu")
	})
	pages.AddPage("init", f.SetBorder(true).SetTitle(" ВХОД "), true, true)
	app.SetRoot(pages, true).Run()
}
