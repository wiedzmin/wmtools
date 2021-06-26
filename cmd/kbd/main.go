package main

import (
	"fmt"
	"sync"

	"github.com/wiedzmin/toolbox/impl"
	"github.com/wiedzmin/toolbox/impl/shell"
	"go.i3wm.org/i3"
	"go.uber.org/zap"
)

const DEFAULT_LAYOUT = "us"

var windows map[int64]string
var wg sync.WaitGroup

var logger *zap.Logger

func getFocusedContainer() (*i3.Node, error) {
	tree, err := i3.GetTree()
	if err != nil {
		return nil, err
	}
	con := tree.Root.FindFocused(func(n *i3.Node) bool {
		return n.Focused && n.Type == i3.Con
	})
	if con == nil {
		return nil, fmt.Errorf("could not find a focused container")
	}
	return con, nil
}

func handleBindings() {
	l := logger.Sugar()
	defer wg.Done()
	recv := i3.Subscribe(i3.BindingEventType)
	defer recv.Close()
	for recv.Next() {
		ev := recv.Event().(*i3.BindingEvent)
		if ev.Binding.Command == "nop" && ev.Binding.Symbol == "backslash" {
			_, _ = shell.ShellCmd("xkb-switch -n", nil, nil, false, false)
			layout, _ := shell.ShellCmd("xkb-switch", nil, nil, true, false)
			con, _ := getFocusedContainer()
			windows[con.Window] = *layout
			l.Debugw("[handleBindings]", "layout", layout, "window",
				fmt.Sprintf("'%s'/'%s'", con.WindowProperties.Title, con.WindowProperties.Class))
		}
	}
}

func handleWindows() {
	l := logger.Sugar()
	defer wg.Done()
	recv := i3.Subscribe(i3.WindowEventType)
	defer recv.Close()
	for recv.Next() {
		ev := recv.Event().(*i3.WindowEvent)
		switch ev.Change {
		case "focus":
			layout, ok := windows[ev.Container.Window]
			if !ok {
				layout = DEFAULT_LAYOUT
			}
			l.Debugw("[handleWindows]", "event", "focus",
				"window", fmt.Sprintf("'%s'/'%s'", ev.Container.WindowProperties.Title, ev.Container.WindowProperties.Class),
				"layout", layout,
				"firsttime", !ok)
			_, _ = shell.ShellCmd(fmt.Sprintf("xkb-switch -s %s", layout), nil, nil, false, false)
		case "close":
			l.Debugw("[handleWindows]", "event", "close",
				"window", fmt.Sprintf("'%s'/'%s'", ev.Container.WindowProperties.Title, ev.Container.WindowProperties.Class))
			delete(windows, ev.Container.Window)
		}
	}
}

func main() {
	logger = impl.NewLogger()
	defer logger.Sync()
	windows = make(map[int64]string)
	impl.EnsureBinary("xkb-switch", *logger)
	wg.Add(1)
	go handleWindows()
	wg.Add(1)
	go handleBindings()
	wg.Wait()
}
