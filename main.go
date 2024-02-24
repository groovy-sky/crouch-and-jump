package main

import (
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/gdamore/tcell"
)

// Icons for the player, obstacles, hearts, etc.
var (
	PlayerIcon   = '▆'
	ObstacleIcon = '▣'
	HeartIcon    = '♥'
	GroundIcon   = '▒'
	BorderIcon   = '░'
)

// GameHandler holds the high score and creates new games.
type GameHandler struct {
	highScore int
	done      chan struct{}
}

// NewGameHandler creates a new game handler.
func NewGameHandler() *GameHandler {
	return &GameHandler{
		highScore: 0,
		done:      make(chan struct{}),
	}
}

// Game represents the game state.
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
	highScore     int
}

// Object represents an object in the game.
type Object struct {
	Coordinates
	Type int
}

// Coordinates represents the position of an object.
type Coordinates struct {
	xPos      int
	tickCount int
	yPos      int
}

func (g *Game) IntroScreen() {
	g.screen.Clear()

	// Draw the intro screen
	introText := []string{
		"Welcome to the Game!",
		"Controls:",
		"  Up arrow: Jump",
		"  Down arrow: Crouch",
		"  Escape: Quit game",
		fmt.Sprintf("High score: %d", g.highScore),
		"Press any key to start the game...",
	}

	for i, line := range introText {
		for j, r := range line {
			g.screen.SetContent(j+2, i+2, r, nil, tcell.StyleDefault)
		}
	}
	g.screen.Show()

	// Wait for any key press
	for {
		ev := g.screen.PollEvent()
		switch event := ev.(type) {
		case *tcell.EventKey:
			switch event.Key() {
			case tcell.KeyEscape:
				close(g.quit)
				g.screen.Fini()
				fmt.Println("Game ended from intro screen.")
				os.Exit(0)
			default:
				return
			}
		}
	}
}

// NewGame creates a new game with the given board width and height.
func (gh *GameHandler) NewGame(boardWidth, boardHeight int, borderIcon rune) (*Game, error) {
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
		playerPos:   2,
		lives:       3,
		objects:     objects,
		quit:        make(chan struct{}),
		events:      events,
		boardWidth:  boardWidth,
		boardHeight: boardHeight,
		borderIcon:  borderIcon,
		highScore:   gh.highScore,
	}, nil
}

func (g *Game) DrawBox() {
	// Draw full height vertical border column at the beginning and end of the game screen
	for y := 0; y < g.boardHeight; y++ {
		g.screen.SetContent(0, y, BorderIcon, nil, tcell.StyleDefault)
		g.screen.SetContent(g.boardWidth-1, y, BorderIcon, nil, tcell.StyleDefault)
	}
	g.screen.Show()
}

func (g *Game) DrawLogo() {
	// Define the logo as a slice of strings
	logo := []string{
		"  _____        __  __ _       ",
		" ᏟᏒᎧᏟᎻ ᎪNᎠ ᎫUᎷᎮ     ",
		"| |  __  __ _| |_| |_| |_   _ ",
	}

	// Calculate the starting point to center the logo
	startX := (g.boardWidth - len(logo[0])) / 2
	startY := (g.boardHeight - len(logo)) / 2

	// Draw the logo inside the box
	for y, line := range logo {
		for x, ch := range line {
			g.screen.SetContent(startX+x, startY+y, ch, nil, tcell.StyleDefault)
		}
	}
	g.screen.Show()

	// Wait for a few seconds
	time.Sleep(3 * time.Second)
}

// Draw draws the game to the screen.
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

	// Draw the ground

	for x := 0; x < g.boardWidth; x++ {
		g.screen.SetContent(x, g.boardHeight, GroundIcon, nil, tcell.StyleDefault)
	}
	// Draw top border
	for x := 0; x < g.boardWidth; x++ {
		g.screen.SetContent(x, 0, BorderIcon, nil, tcell.StyleDefault)
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
		g.screen.SetContent(x, g.boardHeight+1, BorderIcon, nil, tcell.StyleDefault)
	}
	// Draw left border
	for y := 0; y < g.boardHeight+2; y++ {
		g.screen.SetContent(0, y, BorderIcon, nil, tcell.StyleDefault)
	}
	// Draw right border
	for y := 0; y < g.boardHeight+2; y++ {
		g.screen.SetContent(g.boardWidth, y, BorderIcon, nil, tcell.StyleDefault)
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
				o.xPos = g.boardWidth
				o.tickCount = rand.Intn(g.boardWidth)
				if o.Type == 0 { // If not a heart, then it's an obstacle
					g.score++ // Increment the score only when the player avoids an obstacle
				} else if o.Type == 1 { // heart
					if g.lives < 3 {
						g.lives++
					}
				}
			}
		}
		if o.xPos == g.playerPos-1 {
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
			} else if o.Type == 1 { // heart
				if g.lives < 3 {
					g.lives++
				}
			}
		}
	}
}

// Handles keyboard events.
func (g *Game) HandleEvent(e tcell.Event, gh *GameHandler) {
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
			close(gh.done) // signal that the application should exit
		}
	}
}

// Run starts the game loop.
func (g *Game) Run(gh *GameHandler) {
	g.DrawBox()
	g.DrawLogo()
	g.IntroScreen()
	ticker := time.NewTicker(80 * time.Millisecond)
	for {
		select {
		case e := <-g.events:
			g.HandleEvent(e, gh)
		case <-ticker.C:
			g.Update()
			g.Draw()
		case <-g.quit:
			if g.score > g.highScore {
				g.highScore = g.score
				gh.highScore = g.score // update the high score in the game handler
			}
			g.screen.Fini()
			fmt.Println("Game over! Your score: ", g.score)
			return
		}
	}
}

func main() {
	// Draws intro logo with size of boardWidth and boardHeight and waits 3 seconds
	// Then starts the game

	gh := NewGameHandler() // create a new game handler
	for {
		select {
		case <-gh.done:
			return // exit the application when done signal is received
		default:
			boardWidth := 40
			boardHeight := 6
			game, err := gh.NewGame(boardWidth, boardHeight, ObstacleIcon) // create a new game using the game handler
			if err != nil {
				panic(err)
			}
			game.Run(gh) // run the game using the game handler
		}
	}
}
