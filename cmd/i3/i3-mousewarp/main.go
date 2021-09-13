package main

import (
	"fmt"
	"sync"

	"github.com/wiedzmin/toolbox/impl"
	"github.com/wiedzmin/toolbox/impl/shell"
	"go.i3wm.org/i3"
	"go.uber.org/zap"
)

var wg sync.WaitGroup
var logger *zap.Logger

func handleWindows() {
	defer wg.Done()
	recv := i3.Subscribe(i3.WindowEventType)
	defer recv.Close()
	for recv.Next() {
		ev := recv.Event().(*i3.WindowEvent)
		switch ev.Change {
		case "focus":
			// TODO: parameterize windows fraction sizes below (/3)
			cursorX := ev.Container.Rect.X + ev.Container.Rect.Width/3
			cursorY := ev.Container.Rect.Y + ev.Container.Rect.Height/3
			_, _ = shell.ShellCmd(fmt.Sprintf("xdotool mousemove %d %d", cursorX, cursorY), nil, nil, false, false)
		}
	}
}

func main() {
	logger = impl.NewLogger()
	defer logger.Sync()
	impl.EnsureBinary("xdotool", *logger)
	wg.Add(1)
	go handleWindows()
	wg.Wait()
}
