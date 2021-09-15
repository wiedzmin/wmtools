package main

import (
	"fmt"
	"os"
	"sync"

	"github.com/urfave/cli/v2"
	"github.com/wiedzmin/toolbox/impl"
	"github.com/wiedzmin/toolbox/impl/shell"
	"go.i3wm.org/i3"
	"go.uber.org/zap"
)

var wg sync.WaitGroup
var logger *zap.Logger

func handleWindows(ctx *cli.Context) {
	defer wg.Done()
	recv := i3.Subscribe(i3.WindowEventType)
	defer recv.Close()
	for recv.Next() {
		ev := recv.Event().(*i3.WindowEvent)
		switch ev.Change {
		case "focus":
			cursorX := float64(ev.Container.Rect.X) + float64(ev.Container.Rect.Width)*ctx.Float64("x-frac")
			cursorY := float64(ev.Container.Rect.Y) + float64(ev.Container.Rect.Height)*ctx.Float64("y-frac")
			_, _ = shell.ShellCmd(fmt.Sprintf("xdotool mousemove %f %f", cursorX, cursorY), nil, nil, false, false)
		}
	}
}

func perform(ctx *cli.Context) error {
	wg.Add(1)
	go handleWindows(ctx)
	wg.Wait()
	return nil
}

func createCLI() *cli.App {
	app := cli.NewApp()
	app.Name = "i3-mousewarp"
	app.Usage = "Warps mouse cursor to the position of recently activated window"
	app.Description = "i3-desktops"
	app.Version = "0.0.1#master"

	app.Flags = []cli.Flag{
		&cli.Float64Flag{
			Name:     "x-frac",
			Aliases:  []string{"x"},
			Usage:    "X fraction for cursor placement between 0 and 1, 0.5 denotes window center",
			Value:    0.5,
			Required: false,
		},
		&cli.Float64Flag{
			Name:     "y-frac",
			Aliases:  []string{"y"},
			Usage:    "Y fraction for cursor placement between 0 and 1, 0.5 denotes window center",
			Value:    0.5,
			Required: false,
		},
	}
	app.Action = perform
	return app
}

func main() {
	logger = impl.NewLogger()
	defer logger.Sync()
	l := logger.Sugar()
	impl.EnsureBinary("xdotool", *logger)
	app := createCLI()
	err := app.Run(os.Args)
	if err != nil {
		l.Errorw("[main]", "err", err)
	}
}
