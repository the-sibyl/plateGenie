package plateGenie

import (
	"github.com/the-sibyl/goLCD20x4"
	"time"
)

// Debounce interval in microseconds
const debounceTime = 160000

type Menu struct {
	lcd             *goLCD20x4.LCD20x4
	firstMenuItem   *MenuItem
	lastMenuItem    *MenuItem
	currentMenuItem *MenuItem
}

func CreateMenu(lcd *goLCD20x4.LCD20x4) *Menu {
	var m Menu
	m.lcd = lcd
	return &m
}

func (m *Menu) Button1Pressed() {
	m.currentMenuItem = m.currentMenuItem.prev
	m.Repaint()
	time.Sleep(time.Microsecond * debounceTime)
}

func (m *Menu) Button2Pressed() {
	m.currentMenuItem.action <- 1
	time.Sleep(time.Microsecond * debounceTime)
}

func (m *Menu) Button3Pressed() {
	m.currentMenuItem.action <- 2
	time.Sleep(time.Microsecond * debounceTime)
}

func (m *Menu) Button4Pressed() {
	m.currentMenuItem = m.currentMenuItem.next
	m.Repaint()
	time.Sleep(time.Microsecond * debounceTime)
}

func (m *Menu) Repaint() {
	m.lcd.WriteLineCentered(m.currentMenuItem.Name, 1)
	m.lcd.WriteLineCentered(m.currentMenuItem.Units, 2)
	m.lcd.WriteLineCentered(m.currentMenuItem.Values, 3)
	m.lcd.WriteLine(m.currentMenuItem.Adjustments, 4)
}

type MenuItem struct {
	Name        string
	Units       string
	Values      string
	Adjustments string
	adj1        string
	adj2        string
	// A channel correpsonding to the soft key pressed (1 or 2)
	action chan int
	prev   *MenuItem
	next   *MenuItem
}

// Two adjustents per screen
// Seven-character limit per adjustment
func (m *Menu) AddMenuItem(name string, units string, values string, adj1 string, adj2 string) *MenuItem {
	var mi MenuItem

	mi.Name = name
	mi.Units = units
	mi.Values = values
	// Add a full 7 characters of padding to the end in case the string is empty
	adj1 += "       "
	adj2 += "       "
	mi.adj1 = adj1[0:7]
	mi.adj2 = adj2[0:7]

	mi.FormatAdjustmentsString()

	// Update the links in the menu
	if m.firstMenuItem == nil {
		m.firstMenuItem = &mi
	}

	if m.lastMenuItem == nil {
		m.lastMenuItem = &mi
	}

	if m.currentMenuItem == nil {
		m.currentMenuItem = &mi
		m.Repaint()
	}

	// Update links for the previous menu item
	m.lastMenuItem.next = &mi

	// Update links for this menu item
	mi.prev = m.lastMenuItem
	m.lastMenuItem = &mi
	mi.next = m.firstMenuItem

	// Update the links in the first and last menu items
	m.firstMenuItem.prev = &mi
	m.lastMenuItem.next = m.firstMenuItem

	mi.action = make(chan int)

	return &mi
}

// Output channels for the two user-defined softkey actions. Return a read-only channel.
func (mi *MenuItem) AddAction() (action <-chan int) {
	action = make(<-chan int)
	action = mi.action
	return action
}

// Helper for the last line which has the adjustment text and previous and next screen arrows
func (mi *MenuItem) FormatAdjustmentsString() {
	sc := goLCD20x4.GetSpecialCharacters()

	mi.Adjustments = sc.LeftArrow + " " + mi.adj1 + "  " + mi.adj2 + " " + sc.RightArrow
}
