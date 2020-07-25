package main

import (
	"fmt"
	"os"
	"time"

	"github.com/gdamore/tcell"
	"github.com/mattn/go-runewidth"
)

func createScreen() (tcell.Screen, error) {
	tcell.SetEncodingFallback(tcell.EncodingFallbackASCII)
	s, e := tcell.NewScreen()
	if e != nil {
		return nil, e
	}
	if e = s.Init(); e != nil {
		return nil, e
	}

	s.SetStyle(tcell.StyleDefault.
		Foreground(tcell.ColorWhite).
		Background(tcell.ColorBlack))
	s.EnableMouse()
	s.Clear()
	return s, nil
}

// GameOfLife - a struct for game of life
type GameOfLife struct {
	width      int
	height     int
	evolution  bool
	delay      time.Duration
	field      map[int]map[int]byte
	screen     tcell.Screen
	stop       chan struct{}
	generation int
	stateWidth int
}

func makeNewMap(w int, h int) map[int]map[int]byte {
	m := make(map[int]map[int]byte, h)
	for i := 0; i < h; i++ {
		m[i] = make(map[int]byte, w)
	}
	return m
}

// NewGame - create a new game object
func NewGame(screen tcell.Screen) *GameOfLife {
	w, h := screen.Size()
	w = w - 10
	game := GameOfLife{
		width:      w,
		height:     h,
		delay:      500,
		screen:     screen,
		field:      makeNewMap(w, h),
		stateWidth: 10,
	}

	return &game
}

func (game *GameOfLife) nextCellState(x int, y int) byte {
	neigbors := 0
	for i := -1; i <= 1; i++ {
		for j := -1; j <= 1; j++ {
			nx := x + i
			ny := y + j
			if nx < 0 {
				nx = game.width - 1
			} else if nx == game.width {
				nx = 0
			}
			if ny < 0 {
				ny = game.height - 1
			} else if ny == game.height {
				ny = 0
			}
			neigbors += int(game.field[ny][nx])
		}
	}
	if neigbors == 3 {
		return 1
	} else if neigbors == 4 {
		return game.field[y][x]
	}
	return 0
}
func (game *GameOfLife) tick() {
	nextMap := makeNewMap(game.width, game.height)
	for y := 0; y < game.height; y++ {
		for x := 0; x <= game.width; x++ {
			nextMap[y][x] = game.nextCellState(x, y)
			if nextMap[y][x] != game.field[y][x] {
				game.drawCell(x, y, nextMap[y][x])
			}
		}
	}
	game.field = nextMap
	game.generation++
}

func (game *GameOfLife) reverseCell(x int, y int) {
	game.field[y][x] ^= 1
	game.drawCell(x, y, game.field[y][x])
}

func (game *GameOfLife) switchMode() {
	if game.evolution {
		game.evolution = false
	} else {
		game.evolution = true
	}
}

func (game *GameOfLife) drawCell(x int, y int, state byte) {
	if state == 1 {
		game.screen.SetContent(x+game.stateWidth, y, 'X', nil, tcell.StyleDefault)
	} else {
		game.screen.SetContent(x+game.stateWidth, y, ' ', nil, tcell.StyleDefault)
	}
}

func (game *GameOfLife) drawState() {
	for i := 0; i <= game.stateWidth; i++ {
		for j := 0; j <= 2; j++ {
			game.screen.SetContent(i, 0, ' ', nil, tcell.StyleDefault)
		}
	}
	emitStr(game.screen, 0, 0, tcell.StyleDefault, fmt.Sprintf("%dx%d", game.width, game.height))
	state := "Pause"
	if game.evolution {
		state = "Play"
	}
	emitStr(game.screen, 0, 1, tcell.StyleDefault, state)
	emitStr(game.screen, 0, 2, tcell.StyleDefault, fmt.Sprintf("Gen: %d", game.generation))
}

func (game *GameOfLife) increaseDelay() {
	game.delay += 50
}

func (game *GameOfLife) decreaseDelay() {
	newDelay := game.delay - 50
	if newDelay < 0 {
		newDelay = 0
	}
	game.delay = newDelay
}

func (game *GameOfLife) resize(w int, h int) {
	game.evolution = false
	w = w - game.stateWidth
	game.screen.Fill(' ', tcell.StyleDefault)
	newField := makeNewMap(w, h)
	for y := 0; y < h; y++ {
		if y >= game.height {
			break
		}
		for x := 0; x <= w; x++ {
			newField[y][x] = game.field[y][x]
			if newField[y][x] == 1 {
				game.drawCell(x, y, newField[y][x])
			}
			if x >= game.width {
				break
			}
		}
	}
	game.width = w
	game.height = h
	game.field = newField
}

// Start - start game's loop
func (game *GameOfLife) Start() {
	game.stop = make(chan struct{})
	go func() {
		for {
			ev := game.screen.PollEvent()
			switch ev := ev.(type) {
			case *tcell.EventKey:
				switch ev.Key() {
				case tcell.KeyEscape, tcell.KeyEnter:
					close(game.stop)
					return
				case tcell.KeyLeft:
					game.decreaseDelay()
				case tcell.KeyRight:
					game.increaseDelay()
				case tcell.KeyRune:
					switch ev.Rune() {
					case ' ':
						game.switchMode()
						game.drawState()
						game.screen.Show()
					case 'q', 'Q':
						close(game.stop)
						return
					}
				}
			case *tcell.EventMouse:
				if ev.Buttons()&tcell.Button1 != 0 {
					x, y := ev.Position()
					if x >= game.stateWidth {
						game.reverseCell(x-game.stateWidth, y)
						game.screen.Show()
					}
				}
			case *tcell.EventResize:
				w, h := game.screen.Size()
				game.resize(w, h)
				game.drawState()
				game.screen.Sync()
			}
		}
	}()

	for {
		select {
		case <-game.stop:
			return
		case <-time.After(time.Millisecond * game.delay):
		}
		if game.evolution {
			game.tick()
			game.drawState()
			game.screen.Show()
		}
	}
}

func emitStr(s tcell.Screen, x, y int, style tcell.Style, str string) {
	for _, c := range str {
		var comb []rune
		w := runewidth.RuneWidth(c)
		if w == 0 {
			comb = []rune{c}
			c = ' '
			w = 1
		}
		s.SetContent(x, y, c, comb, style)
		x += w
	}
}

func main() {
	screen, err := createScreen()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
	game := NewGame(screen)

	game.Start()

	screen.Fini()
}
