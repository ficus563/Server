package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
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
	Messages []Message            `json:"messages"`
}

var (
	hero      Player
	app       *tview.Application
	pages     *tview.Pages
	chatBox   *tview.TextView
	worldBox  *tview.TextView
	serverURL = "http://localhost:8080" 
)

func networkLoop() {
	for {
		data, _ := json.Marshal(hero)
		resp, err := http.Post(serverURL+"/sync", "application/json", bytes.NewBuffer(data))
		if err == nil {
			var gs GameState
			if err := json.NewDecoder(resp.Body).Decode(&gs); err == nil {
				app.QueueUpdateDraw(func() {
					chatBox.Clear()
					for _, m := range gs.Messages {
						fmt.Fprintf(chatBox, "[gray]%s[-] [blue]%s:[-] %s\n", m.Time, m.Author, m.Text)
					}
					chatBox.ScrollToEnd()

					worldBox.Clear()
					fmt.Fprintln(worldBox, "[yellow]ИГРОКИ:[-]")
					for name, p := range gs.Players {
						if name == hero.Name { 
							hero.HP = p.HP 
						}
						fmt.Fprintf(worldBox, "• %s (HP: %d)\n", name, p.HP)
					}
				})
			}
			resp.Body.Close()
		}
		time.Sleep(1 * time.Second)
	}
}

func sendMessage(text string) {
	if text == "" { return }
	if strings.HasPrefix(text, "/slap ") {
		target := strings.TrimPrefix(text, "/slap ")
		dmg := 10 + rand.Intn(10)
		payload, _ := json.Marshal(map[string]interface{}{"Target": target, "Damage": dmg, "Attacker": hero.Name})
		go http.Post(serverURL+"/attack", "application/json", bytes.NewBuffer(payload))
		return
	}
	msg, _ := json.Marshal(Message{Author: hero.Name, Text: text})
	go http.Post(serverURL+"/chat", "application/json", bytes.NewBuffer(msg))
}

func main() {
	rand.Seed(time.Now().UnixNano())
	app = tview.NewApplication().EnableMouse(true)
	pages = tview.NewPages()

	// МАКСИМАЛЬНО ПРОСТОЕ ПОЛЕ
	inputName := tview.NewInputField().
		SetLabel("ИМЯ ГЕРОЯ: ").
		SetText("Ficus")

	inputName.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEnter {
			name := inputName.GetText()
			if name == "" { name = "Hero" }
			hero = Player{Name: name, HP: 100}
			showGame()
			pages.SwitchToPage("game")
		}
	})
	inputName.SetBorder(true).SetTitle(" ВВЕДИТЕ ИМЯ И НАЖМИТЕ ENTER ")

	pages.AddPage("login", inputName, true, true) // Никаких флексов, только поле на весь экран

	// Принудительный фокус на поле ввода при старте
	app.SetRoot(pages, true).SetFocus(inputName) 

	if err := app.SetRoot(pages, true).Run(); err != nil { panic(err) }
}

func showGame() {
	chatBox = tview.NewTextView().SetDynamicColors(true).SetWordWrap(true)
	chatBox.SetBorder(true).SetTitle(" ЧАТ (/slap имя - ударить) ")
	
	worldBox = tview.NewTextView().SetDynamicColors(true)
	worldBox.SetBorder(true).SetTitle(" МИР ")

	input := tview.NewInputField().SetLabel("Сообщение: ")
	input.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEnter {
			sendMessage(input.GetText())
			input.SetText("")
		}
	})

	layout := tview.NewFlex().
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(chatBox, 0, 1, false).
			AddItem(input, 3, 1, true), 0, 2, true).
		AddItem(worldBox, 20, 1, false)

	pages.AddPage("game", layout, true, true)
	app.SetFocus(input) // Фокус на чат после входа
	go networkLoop()
}
