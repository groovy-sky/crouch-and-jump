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

type Obstacle struct {
	xPos      int
	tickCount int
	yPos      int
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

	rand.Seed(time.Now().UnixNano())
	obstacles := make([]Obstacle, 4)
	for i := range obstacles {
		tickCount := (i / 2) * (boardWidth / 2) // 2 obstacles in a row, then a gap
		yPos := boardHeight - rand.Intn(2) + 1  // Random yPos between 1 and 2, considering player's height
		obstacles[i] = Obstacle{
			xPos:      boardWidth,
			tickCount: tickCount,
			yPos:      yPos,
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
	yPos := g.boardHeight - g.jumpHeight - 1 // Subtract 1 to make the player one cell higher
	g.screen.SetContent(g.playerPos, yPos, PlayerIcon, nil, tcell.StyleDefault)
	if !g.crouching {
		g.screen.SetContent(g.playerPos, yPos-1, PlayerIcon, nil, tcell.StyleDefault)
	}
	for _, o := range g.obstacles {
		if o.xPos >= 0 {
			g.screen.SetContent(o.xPos, g.boardHeight-1, ObstacleIcon, nil, tcell.StyleDefault) // Subtract 1 to make the obstacle one cell higher
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
				o.xPos = g.boardWidth
				o.tickCount = rand.Intn(g.boardWidth)
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
	boardWidth := 40
	boardHeight := 6
	game, err := NewGame(boardWidth, boardHeight, ObstacleIcon)
	if err != nil {
		panic(err)
	}
	game.Run()
}
