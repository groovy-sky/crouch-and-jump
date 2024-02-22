package main

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/gdamore/tcell"
)

var (
	PlayerIcon   = '█'
	ObstacleIcon = '▣'
)

type Obstacle struct {
	xPos      int
	tickCount int
}

type Game struct {
	screen      tcell.Screen
	playerPos   int
	obstacles   []Obstacle
	score       int
	jumping     bool
	crouching   bool
	jumpHeight  int
	crouchTicks int
	quit        chan struct{}
	events      chan tcell.Event
	boardWidth  int
	boardHeight int
	borderIcon  rune
}

func NewGame(boardWidth, boardHeight int, borderIcon rune) (*Game, error) {
	screen, err := tcell.NewScreen()
	if err != nil {
		return nil, err
	}
	if err := screen.Init(); err != nil {
		return nil, err
	}
	events := make(chan tcell.Event)
	go func() {
		for {
			events <- screen.PollEvent()
		}
	}()

	obstacles := make([]Obstacle, 5)
	for i := range obstacles {
		obstacles[i] = Obstacle{
			xPos:      boardWidth,
			tickCount: rand.Intn(boardWidth),
		}
	}

	return &Game{
		screen:      screen,
		playerPos:   1,
		obstacles:   obstacles,
		quit:        make(chan struct{}),
		events:      events,
		boardWidth:  boardWidth,
		boardHeight: boardHeight,
		borderIcon:  borderIcon,
	}, nil
}

func (g *Game) Draw() {
	g.screen.Clear()
	yPos := g.boardHeight - g.jumpHeight
	g.screen.SetContent(g.playerPos, yPos, PlayerIcon, nil, tcell.StyleDefault)
	if !g.crouching {
		g.screen.SetContent(g.playerPos, yPos-1, PlayerIcon, nil, tcell.StyleDefault)
	}
	for _, o := range g.obstacles {
		if o.xPos >= 0 {
			g.screen.SetContent(o.xPos, g.boardHeight, ObstacleIcon, nil, tcell.StyleDefault)
		}
	}
	// Draw top border
	for x := 0; x < g.boardWidth; x++ {
		g.screen.SetContent(x, 0, g.borderIcon, nil, tcell.StyleDefault)
	}
	// Draw bottom border
	for x := 0; x < g.boardWidth; x++ {
		g.screen.SetContent(x, g.boardHeight+1, g.borderIcon, nil, tcell.StyleDefault)
	}
	// Draw left border
	for y := 0; y < g.boardHeight+2; y++ {
		g.screen.SetContent(0, y, g.borderIcon, nil, tcell.StyleDefault)
	}
	// Draw right border
	for y := 0; y < g.boardHeight+2; y++ {
		g.screen.SetContent(g.boardWidth, y, g.borderIcon, nil, tcell.StyleDefault)
	}
	g.screen.Show()
}
func (g *Game) Update() {
	if g.jumping {
		if g.jumpHeight < 2 {
			g.jumpHeight++
		} else {
			g.jumping = false
		}
	} else if g.jumpHeight > 0 {
		g.jumpHeight--
	}

	if g.crouching && g.crouchTicks > 0 {
		g.crouchTicks--
		if g.crouchTicks == 0 {
			g.crouching = false
		}
	}

	for i := range g.obstacles {
		o := &g.obstacles[i]
		if o.tickCount > 0 {
			o.tickCount--
		} else {
			o.xPos--
			if o.xPos < 0 {
				o.xPos = 80
				o.tickCount = rand.Intn(80)
				g.score++
			}
		}
		if o.xPos == g.playerPos && g.jumpHeight == 0 {
			close(g.quit)
		}
	}
}

func (g *Game) HandleEvent(e tcell.Event) {
	switch ev := e.(type) {
	case *tcell.EventKey:
		if ev.Key() == tcell.KeyUp && !g.jumping && g.jumpHeight == 0 && !g.crouching {
			g.jumping = true
		}
		if ev.Key() == tcell.KeyDown && !g.jumping && g.jumpHeight == 0 {
			g.crouching = true
			g.crouchTicks = 3
		}
		if ev.Key() == tcell.KeyEscape {
			close(g.quit)
		}
	}
}

func (g *Game) Run() {
	ticker := time.NewTicker(200 * time.Millisecond)
	for {
		select {
		case <-g.quit:
			g.screen.Fini()
			fmt.Println("Game over! Your score: ", g.score)
			return
		case e := <-g.events:
			g.HandleEvent(e)
		case <-ticker.C:
			g.Update()
			g.Draw()
		}
	}
}

func main() {
	game, err := NewGame(30, 10, '#')
	if err != nil {
		panic(err)
	}
	game.Run()
}
