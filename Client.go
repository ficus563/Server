package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

// структуры и интерфейсы
type character interface {
	hit_target(target character, target_zone string)
	block_attack(target_zone string) bool
	get_hp() int
	get_name() string
}

type item struct {
	item_type string
	name      string
	attack    int
	defence   int
	heal_hp   int
}

type player struct {
	name      string
	hp        int
	strength  int
	hit       string
	block     string
	inventory []item
	equipment []item
}

type enemy struct {
	name     string
	hp       int
	strength int
	hit      string
	block    string
	loot     *item
}

func (p *player) get_hp() int                        { return p.hp }
func (p *player) get_name() string                   { return p.name }
func (e *enemy) get_hp() int                         { return e.hp }
func (e *enemy) get_name() string                    { return e.name }
func (p *player) block_attack(target_zone string) bool { return p.block == target_zone }
func (e *enemy) block_attack(target_zone string) bool  { return e.block == target_zone }

func (p *player) hit_target(target character, target_zone string) {
	damage := p.strength
	weapon_idx := -1

	for i, it := range p.equipment {
		if it.item_type == "оружие" {
			damage += it.attack
			weapon_idx = i
			break
		}
	}

	switch target_zone {
	case "руки", "ноги":
		damage += 5
	case "живот":
		damage += 10
	case "грудь":
		damage += 15
	case "голова":
		damage += 20
	}

	if weapon_idx != -1 {
		fmt.Printf("Герой %s атаковал оружием '%s' (+%d урона), и оно затупилось/сломалось.\n", p.name, p.equipment[weapon_idx].name, p.equipment[weapon_idx].attack)
		p.equipment = append(p.equipment[:weapon_idx], p.equipment[weapon_idx+1:]...)
	}

	if !target.block_attack(target_zone) {
		has_armor := false
		if t, ok := target.(*player); ok {
			for i := 0; i < len(t.equipment); i++ {
				if t.equipment[i].item_type == "броня" {
					has_armor = true
					t.equipment[i].defence -= damage
					if t.equipment[i].defence <= 0 {
						fmt.Printf("Доспех героя %s полностью разрушен!\n", t.name)
						t.equipment = append(t.equipment[:i], t.equipment[i+1:]...)
					} else {
						fmt.Printf("Доспех героя %s поглотил урон (осталось прочности: %d).\n", t.name, t.equipment[i].defence)
					}
					break
				}
			}
			if !has_armor {
				t.hp -= damage
				fmt.Printf("Герой %s получил %d урона в область: %s.\n", t.name, damage, target_zone)
			}
		} else if t, ok := target.(*enemy); ok {
			t.hp -= damage
			fmt.Printf("Враг %s получил %d урона в область: %s.\n", t.name, damage, target_zone)
		}
	} else {
		fmt.Printf("Удар в область '%s' был успешно парирован!\n", target_zone)
	}
}

func (e *enemy) hit_target(target character, target_zone string) {
	damage := e.strength
	switch target_zone {
	case "руки", "ноги":
		damage += 5
	case "живот":
		damage += 10
	case "грудь":
		damage += 15
	case "голова":
		damage += 20
	}
	if !target.block_attack(target_zone) {
		if t, ok := target.(*player); ok {
			has_armor := false
			for i := 0; i < len(t.equipment); i++ {
				if t.equipment[i].item_type == "броня" {
					has_armor = true
					t.equipment[i].defence -= damage
					if t.equipment[i].defence <= 0 {
						fmt.Printf("Ваш доспех полностью разрушен!\n")
						t.equipment = append(t.equipment[:i], t.equipment[i+1:]...)
					} else {
						fmt.Printf("Ваш доспех поглотил урон врага (осталось прочности: %d).\n", t.equipment[i].defence)
					}
					break
				}
			}
			if !has_armor {
				t.hp -= damage
				fmt.Printf("Враг %s нанес вам %d урона в область: %s!\n", e.name, damage, target_zone)
			}
		}
	} else {
		fmt.Printf("Вы успешно парировали удар врага в область '%s'.\n", target_zone)
	}
}

// вспомогательные функции
func get_zone_name(idx int) string {
	zones := map[int]string{1: "голова", 2: "грудь", 3: "живот", 4: "руки", 5: "ноги"}
	return zones[idx]
}

func get_safe_number(scanner *bufio.Scanner, message string, min, max int) int {
	for {
		fmt.Println(message)
		scanner.Scan()
		text := scanner.Text()
		if text == "exit" {
			return -1
		}
		num, err := strconv.Atoi(text)
		if err == nil && num >= min && num <= max {
			return num
		}
		fmt.Printf("Ошибка. Введите число от %d до %d или 'exit'\n", min, max)
	}
}

func get_valid_name(scanner *bufio.Scanner, prompt string) string {
	for {
		fmt.Println(prompt)
		scanner.Scan()
		name := strings.TrimSpace(scanner.Text())
		if name != "" && name != "exit" {
			return name
		}
		fmt.Println("Имя не может быть пустым или 'exit'.")
	}
}

func show_inventory(inv []item) {
	fmt.Println("I. Оружие")
	for i, it := range inv {
		if it.item_type == "оружие" {
			fmt.Printf("\t%d. %s (%d ед. урона)\n", i+1, it.name, it.attack)
		}
	}
	fmt.Println("II. Броня")
	for i, it := range inv {
		if it.item_type == "броня" {
			fmt.Printf("\t%d. %s (%d ед. прочности)\n", i+1, it.name, it.defence)
		}
	}
	fmt.Println("III. Зелья и еда")
	for i, it := range inv {
		if it.item_type == "хилка" {
			fmt.Printf("\t%d. %s (восстанавливает %d хп)\n", i+1, it.name, it.heal_hp)
		}
	}
}

func show_and_choose_inventory(scanner *bufio.Scanner, inv []item, action string) int {
	for {
		show_inventory(inv)
		fmt.Printf("Введите номер предмета чтобы %s (или 'exit' для отмены): \n", action)
		scanner.Scan()
		text := scanner.Text()
		if text == "exit" {
			return -1
		}
		num, err := strconv.Atoi(text)
		if err == nil && num >= 1 && num <= len(inv) {
			return num - 1
		}
		fmt.Println("Ошибка! Неверный номер предмета.")
	}
}

func get_random_loot() *item {
	loot_table := []item{
		{"оружие", "Ржавый меч", 15, 0, 0},
		{"оружие", "Кинжал разбойника", 10, 0, 0},
		{"оружие", "Костяная дубина", 12, 0, 0},
		{"оружие", "Пылающий секира", 30, 0, 0},
		{"броня", "Старый плащ", 0, 10, 0},
		{"броня", "Кольчуга стражника", 0, 60, 0},
		{"броня", "Кожаный доспех", 0, 50, 0},
		{"хилка", "Большое зелье исцеления", 0, 0, 30},
		{"хилка", "Кусок жареного мяса", 0, 0, 20},
		{"хилка", "Эльфийский хлеб", 0, 0, 15},
	}
	selected := loot_table[rand.Intn(len(loot_table))]
	return &selected
}

func local_player_menu(scanner *bufio.Scanner, p *player) bool {
	move_done := false
	for !move_done {
		fmt.Printf("\n--- ХОД ГЕРОЯ %s (%d HP) ---\n", p.name, p.hp)
		fmt.Println("1. Атаковать\n2. Экипировать\n3. Показать инвентарь\n4. Снять предмет")
		choice := get_safe_number(scanner, "Ваш выбор:", 1, 4)
		if choice == -1 {
			return false
		}

		switch choice {
		case 1:
			hit := get_safe_number(scanner, "Куда бьем? (1 - голова, 2 - грудь, 3 - живот, 4 - руки, 5 - ноги):", 1, 5)
			block := get_safe_number(scanner, "Что защищаем? (1 - голова, 2 - грудь, 3 - живот, 4 - руки, 5 - ноги):", 1, 5)
			p.hit, p.block = get_zone_name(hit), get_zone_name(block)
			move_done = true
		case 2:
			if len(p.inventory) == 0 {
				fmt.Println("Сумка пуста.")
				continue
			}
			idx := show_and_choose_inventory(scanner, p.inventory, "экипировать/использовать")
			if idx != -1 {
				it := p.inventory[idx]
				if it.item_type == "хилка" {
					p.hp += it.heal_hp
					fmt.Printf("Вы использовали '%s' и восстановили %d хп.\n", it.name, it.heal_hp)
				} else {
					p.equipment = append(p.equipment, it)
					fmt.Printf("Вы экипировали '%s'.\n", it.name)
				}
				p.inventory = append(p.inventory[:idx], p.inventory[idx+1:]...)
			}
		case 3:
			fmt.Printf("HP: %d\n--- Инвентарь ---\n", p.hp)
			show_inventory(p.inventory)
			fmt.Println("--- Экипировано ---")
			show_inventory(p.equipment)
		case 4:
			if len(p.equipment) == 0 {
				fmt.Println("Ничего не надето.")
				continue
			}
			idx := show_and_choose_inventory(scanner, p.equipment, "снять")
			if idx != -1 {
				it := p.equipment[idx]
				p.inventory = append(p.inventory, it)
				p.equipment = append(p.equipment[:idx], p.equipment[idx+1:]...)
				fmt.Printf("Предмет '%s' убран в сумку.\n", it.name)
			}
		}
	}
	return true
}

// локальные режимы
type scenario struct {
	chapter_text string
	enemy_name   string
	enemy_hp     int
}

func play_story(scanner *bufio.Scanner) {
	fmt.Println("Нашего главного героя зовут Артур. Он наемник, ищущий славы в Темных Землях...")
	p := &player{name: "Артур", hp: 200, strength: 10, inventory: []item{
		{item_type: "оружие", name: "Короткий меч", attack: 10},
		{item_type: "хилка", name: "Малое зелье", heal_hp: 10},
	}}

	chapters := []scenario{
		{
			chapter_text: "Глава 1: Темный лес.\nАртур углубился в чащу Темного леса, надеясь сократить путь до ближайшего города.\nДеревья смыкались над головой, не пропуская свет луны.\nВнезапно из кустов выпрыгнуло зеленокожее существо с кривым ножом в руках.\nЭто был Гоблин-разведчик, жаждущий легкой наживы.",
			enemy_name:   "Гоблин-разведчик", enemy_hp: 100,
		},
		{
			chapter_text: "Глава 2: Заброшенный тракт.\nПобедив гоблина, Артур продолжил путь. Лес расступился, показав древний, заброшенный тракт.\nОднако покой длился недолго. Огромная фигура преградила ему путь.\nЭто был Орк-мародер, закованный в грубую сталь. Он издал яростный рев и бросился в атаку.",
			enemy_name:   "Орк-мародер", enemy_hp: 150,
		},
		{
			chapter_text: "Глава 3: Руины древнего замка.\nТракт привел Артура к мрачным руинам.\nИменно здесь, по слухам, обитал источник местного зла.\nИз тени разрушенных врат вышел Темный Рыцарь, чьи глаза светились потусторонним пламенем.\nЕго двуручный меч жаждал крови живых. Последний бой начался.",
			enemy_name:   "Темный Рыцарь", enemy_hp: 200,
		},
	}

	for i, scen := range chapters {
		fmt.Printf("\n=== %s ===\n", scen.chapter_text)
		e := &enemy{name: scen.enemy_name, hp: scen.enemy_hp, strength: 10}
		if i < 2 {
			e.loot = get_random_loot()
		}

		for p.hp > 0 && e.hp > 0 {
			if !local_player_menu(scanner, p) {
				return
			}

			e.hit = get_zone_name(rand.Intn(5) + 1)
			e.block = get_zone_name(rand.Intn(5) + 1)

			p.hit_target(e, p.hit)
			e.hit_target(p, e.hit)
			fmt.Printf("=== Итоги раунда: Здоровье %s: %d HP | Здоровье %s: %d HP ===\n", p.name, p.hp, e.name, e.hp)
		}

		if p.hp <= 0 {
			fmt.Printf("К сожалению, вы пали в бою... Тьма поглотила Темные Земли.\n")
			return
		} else {
			fmt.Printf("Вы повергли врага: %s!\n", e.name)
			if e.loot != nil {
				fmt.Printf("Добыча: вы нашли '%s'!\n", e.loot.name)
				p.inventory = append(p.inventory, *e.loot)
			}
		}
	}
	fmt.Println("Эпилог.\nПосле тяжелейших битв Артур очистил руины от зла.\nМестные жители наконец-то смогли вздохнуть спокойно.\nИмя наемника стало легендой, а его путь только начинался. Конец.")
}

func play_hotseat(scanner *bufio.Scanner) {
	fmt.Println("Введите имя Героя 1:")
	p1_name := get_valid_name(scanner, "")
	fmt.Println("Введите имя Героя 2:")
	p2_name := get_valid_name(scanner, "")

	p1 := &player{name: p1_name, hp: 100, strength: 10, inventory: []item{
		{item_type: "оружие", name: "Железный топор", attack: 10},
		{item_type: "хилка", name: "Целебная трава", heal_hp: 15},
	}}
	p2 := &player{name: p2_name, hp: 100, strength: 10, inventory: []item{
		{item_type: "оружие", name: "Рыцарский меч", attack: 12},
		{item_type: "хилка", name: "Отвар знахаря", heal_hp: 15},
	}}

	for p1.hp > 0 && p2.hp > 0 {
		if !local_player_menu(scanner, p1) {
			return
		}
		fmt.Println("\n\n\n\n[Передайте клавиатуру второму игроку]")
		if !local_player_menu(scanner, p2) {
			return
		}

		p1.hit_target(p2, p1.hit)
		p2.hit_target(p1, p2.hit)
		fmt.Printf("=== Итоги раунда: Здоровье %s: %d HP | Здоровье %s: %d HP ===\n", p1.name, p1.hp, p2.name, p2.hp)
	}

	if p1.hp <= 0 && p2.hp <= 0 {
		fmt.Println("Бой окончился ничьей! Оба героя пали.")
	} else if p1.hp <= 0 {
		fmt.Printf("Победил герой %s!\n", p2.name)
	} else if p2.hp <= 0 {
		fmt.Printf("Победил герой %s!\n", p1.name)
	}
}

// сетевой режим
func play_network_client(scanner *bufio.Scanner) {
	fmt.Println("Введите URL сервера (для GitHub Codespaces формат: https://<название-вашего-codespace>-8080.app.github.dev):")
	scanner.Scan()
	url := strings.TrimSpace(scanner.Text())

	fmt.Println("Введите ваше имя:")
	my_name := get_valid_name(scanner, "")
	http.Post(url, "text/plain", bytes.NewBufferString("NAME:"+my_name))

	me := &player{name: my_name, hp: 100, strength: 10}
	me.inventory = []item{
		{item_type: "оружие", name: "Искрящийся клинок", attack: 30},
		{item_type: "оружие", name: "Кинжал ассасина", attack: 20},
		{item_type: "броня", name: "Мифриловая кольчуга", defence: 50},
		{item_type: "хилка", name: "Флакон с кровью дракона", heal_hp: 20},
	}

	last_log_len := 0
	for {
		resp, _ := http.Get(url)
		body, _ := io.ReadAll(resp.Body)
		logs := strings.Split(strings.TrimSpace(string(body)), "\n")

		if len(logs) > last_log_len {
			for i := last_log_len; i < len(logs); i++ {
				fmt.Println(logs[i])
				if strings.Contains(logs[i], "Ожидание хода противника...") {

					move_done := false
					for !move_done {
						fmt.Println("\n1. Атаковать\n2. Экипировать\n3. Показать инвентарь\n4. Снять предмет\n5. Написать сообщение")
						choice := get_safe_number(scanner, "Ваш выбор:", 1, 5)
						if choice == -1 {
							http.Post(url, "text/plain", bytes.NewBufferString("exit"))
							os.Exit(0)
						}

						switch choice {
						case 1:
							h := get_safe_number(scanner, "Куда бьем? (1 - голова, 2 - грудь, 3 - живот, 4 - руки, 5 - ноги):", 1, 5)
							b := get_safe_number(scanner, "Что защищаем? (1 - голова, 2 - грудь, 3 - живот, 4 - руки, 5 - ноги):", 1, 5)
							zones := map[int]string{1: "голова", 2: "грудь", 3: "живот", 4: "руки", 5: "ноги"}
							move_str := zones[h] + ":" + zones[b]

							for idx, it := range me.equipment {
								if it.item_type == "оружие" {
									me.equipment = append(me.equipment[:idx], me.equipment[idx+1:]...)
									break
								}
							}
							http.Post(url, "text/plain", bytes.NewBufferString(move_str))
							move_done = true
						case 2:
							if len(me.inventory) == 0 {
								fmt.Println("Сумка пуста.")
								continue
							}
							idx := show_and_choose_inventory(scanner, me.inventory, "экипировать")
							if idx != -1 {
								selected_item := me.inventory[idx]
								if selected_item.item_type == "хилка" {
									me.hp += selected_item.heal_hp
									http.Post(url, "text/plain", bytes.NewBufferString(fmt.Sprintf("HEAL:%d", selected_item.heal_hp)))
								} else {
									me.equipment = append(me.equipment, selected_item)
									val := selected_item.attack
									if selected_item.item_type == "броня" {
										val = selected_item.defence
									}
									http.Post(url, "text/plain", bytes.NewBufferString(fmt.Sprintf("EQUIP:%s:%s:%d", selected_item.item_type, selected_item.name, val)))
								}
								me.inventory = append(me.inventory[:idx], me.inventory[idx+1:]...)
								fmt.Println("Действие выполнено.")
							}
						case 3:
							fmt.Printf("HP: %d\n--- Инвентарь ---\n", me.hp)
							show_inventory(me.inventory)
							fmt.Println("--- Экипировано ---")
							show_inventory(me.equipment)
						case 4:
							if len(me.equipment) == 0 {
								fmt.Println("Ничего не надето.")
								continue
							}
							idx := show_and_choose_inventory(scanner, me.equipment, "снять")
							if idx != -1 {
								selected_item := me.equipment[idx]
								me.inventory = append(me.inventory, selected_item)
								me.equipment = append(me.equipment[:idx], me.equipment[idx+1:]...)
								http.Post(url, "text/plain", bytes.NewBufferString("UNEQUIP:"+selected_item.name))
								fmt.Println("Предмет убран в сумку.")
							}
						case 5:
							fmt.Println("Введите сообщение:")
							scanner.Scan()
							msg := scanner.Text()
							http.Post(url, "text/plain", bytes.NewBufferString("CHAT:"+msg))
						}
					}
				}
			}
			last_log_len = len(logs)
		}
		if strings.Contains(string(body), "ИГРА ОКОНЧЕНА") {
			break
		}
		time.Sleep(1 * time.Second)
	}
}

// главная функция
func main() {
	rand.Seed(time.Now().UnixNano())
	scanner := bufio.NewScanner(os.Stdin)

	for {
		fmt.Println("\n----- ГЛАВНОЕ МЕНЮ -----")
		fmt.Println("1. Одиночная кампания (сюжет)")
		fmt.Println("2. Дуэль за одним ПК (Hotseat)")
		fmt.Println("3. Сетевой PvP (Клиент)")

		choice := get_safe_number(scanner, "Ваш выбор:", 1, 3)

		if choice == 1 {
			play_story(scanner)
		} else if choice == 2 {
			play_hotseat(scanner)
		} else if choice == 3 {
			play_network_client(scanner)
			break
		}
	}
}
