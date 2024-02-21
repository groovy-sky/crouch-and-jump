package main

import (
	"fmt"
	"time"

	"github.com/gdamore/tcell"
)

type Game struct {
	screen      tcell.Screen
	playerPos   int
	obstacle    int
	score       int
	jumping     bool
	crouching   bool
	jumpHeight  int
	crouchTicks int
	quit        chan struct{}
	events      chan tcell.Event
}

func NewGame() (*Game, error) {
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
	return &Game{
		screen:    screen,
		playerPos: 1,
		obstacle:  80,
		quit:      make(chan struct{}),
		events:    events,
	}, nil
}

func (g *Game) Draw() {
	g.screen.Clear()
	yPos := 20 - g.jumpHeight
	g.screen.SetContent(g.playerPos, yPos, 'P', nil, tcell.StyleDefault)
	if !g.crouching {
		g.screen.SetContent(g.playerPos, yPos-1, 'P', nil, tcell.StyleDefault)
	}
	g.screen.SetContent(g.obstacle, 20, 'O', nil, tcell.StyleDefault)
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

	g.obstacle--
	if g.obstacle < 0 {
		g.obstacle = 80
		g.score++
	}
	if g.obstacle == g.playerPos && g.jumpHeight == 0 {
		close(g.quit)
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
	game, err := NewGame()
	if err != nil {
		panic(err)
	}
	game.Run()
}
