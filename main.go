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
	HeartIcon    = '♥'
)

type Game struct {
	screen        tcell.Screen
	lives         int
	playerPos     int
	objects       []Object
	score         int
	jumping       bool
	crouching     bool
	jumpHeight    int
	crouchTicks   int
	objectCounter int
	quit          chan struct{}
	events        chan tcell.Event
	boardWidth    int
	boardHeight   int
	borderIcon    rune
}

type Object struct {
	Coordinates
	Type int
}

type Coordinates struct {
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
	objects := make([]Object, 8)
	for i := range objects {
		tickCount := (i / 2) * (boardWidth / 2)
		yPos := rand.Intn(2) + 1
		objType := i % 2 // objects with even index will be obstacles, odd index will be hearts
		objects[i] = Object{
			Coordinates: Coordinates{
				xPos:      boardWidth,
				tickCount: tickCount,
				yPos:      yPos,
			},
			Type: objType,
		}
	}

	return &Game{
		screen:      screen,
		playerPos:   1,
		lives:       3,
		objects:     objects,
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
	for _, o := range g.objects {
		if o.xPos >= 0 {
			if o.Type == 0 { // obstacle
				g.screen.SetContent(o.xPos, g.boardHeight-o.yPos, ObstacleIcon, nil, tcell.StyleDefault)
			} else { // heart
				g.screen.SetContent(o.xPos, g.boardHeight-o.yPos, HeartIcon, nil, tcell.StyleDefault)
			}
		}
	}
	// Draw top border
	for x := 0; x < g.boardWidth; x++ {
		g.screen.SetContent(x, 0, g.borderIcon, nil, tcell.StyleDefault)
	}

	// Draw score at the top row
	scoreStr := fmt.Sprintf("Score: %d", g.score)
	for i, r := range scoreStr {
		g.screen.SetContent(i+2, 1, r, nil, tcell.StyleDefault)
	}

	// Draw lives symbols at the top row in the right corner
	for i := 0; i < g.lives; i++ {
		g.screen.SetContent(g.boardWidth-4-i, 1, HeartIcon, nil, tcell.StyleDefault)
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

	for i := range g.objects {
		o := &g.objects[i]
		if o.tickCount > 0 {
			o.tickCount--
		} else {
			o.xPos--
			if o.xPos < 0 {
				g.objectCounter++

				// Every 10 obstacles, replace obstacle with a heart
				if g.objectCounter == 10 && o.Type == 0 {
					yPos := rand.Intn(2) + 1
					g.objects = append(g.objects, Object{
						Coordinates: Coordinates{
							xPos:      g.boardWidth,
							tickCount: rand.Intn(g.boardWidth),
							yPos:      yPos,
						},
						Type: 1, // 1 represents a heart
					})
					g.objectCounter = 0 // Reset the object counter
					o.xPos = -1         // Move the object out of the board
				} else if o.Type == 0 { // If not a heart, then it's an obstacle
					o.xPos = g.boardWidth
					o.tickCount = rand.Intn(g.boardWidth)
					g.score++
				}
			}
		}
		if o.xPos == g.playerPos {
			if o.Type == 0 { // obstacle
				if g.crouching && o.yPos > 1 {
					continue
				}
				if g.jumpHeight <= (o.yPos - 1) {
					g.lives--
					if g.lives == 0 {
						close(g.quit)
					}
				}
			} else if o.Type == 1 && g.lives < 3 { // heart
				g.lives++
				o.xPos = -1 // move the heart out of the board
			}
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
	ticker := time.NewTicker(80 * time.Millisecond)
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
