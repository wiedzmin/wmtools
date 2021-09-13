package main

import (
	"fmt"
	"os"
	"sync"

	"github.com/urfave/cli/v2"
	"github.com/wiedzmin/toolbox/impl"
	"github.com/wiedzmin/toolbox/impl/shell"
	"github.com/wiedzmin/toolbox/impl/xserver"
	"go.i3wm.org/i3"
	"go.uber.org/zap"
)

var (
	logger *zap.Logger
	wg     sync.WaitGroup

	rules      *xserver.WindowRules
	workspaces *xserver.Workspaces
)

func populateMetadata() {
	l := logger.Sugar()
	var err error
	rules, err = xserver.WindowRulesFromRedis("wm/window_rules")
	if err != nil {
		l.Warnw("[prepare]", "err", err)
		os.Exit(1)
	}
	workspaces, err = xserver.WorkspacesFromRedis("wm/workspaces")
	if err != nil {
		l.Warnw("[prepare]", "err", err)
		os.Exit(1)
	}
}

// FIXME: wrong desktop indices are used
func processWindows() error {
	wsXIndexByWorkspaceName := make(map[string]string)
	l := logger.Sugar()
	x, err := xserver.NewX()
	if err != nil {
		l.Warnw("[prepare]", "err", err)
		os.Exit(1)
	}
	windows, err := x.ListWindows()
	if err != nil {
		return err
	}
	for _, win := range windows {
		traits, err := x.GetWindowTraits(&win)
		if err != nil {
			l.Warnw("[processWindows]", "err", err)
			return err
		}
		r, err := rules.MatchTraits(*traits)
		if err != nil {
			l.Warnw("[processWindows]", "err", err)
			continue
		}
		if r != nil {
			ruleWSIndex, ok := wsXIndexByWorkspaceName[r.Desktop]
			l.Debugw("[processWindows]", "window", fmt.Sprintf("%d", win), "rule", r)
			if !ok {
				l.Warnw("[processWindows]", "unknown rule desktop", r.Desktop)
				continue
			} else {
				l.Debugw("[processWindows]", "ruleWSIndex", ruleWSIndex)
				_, err = shell.ShellCmd(fmt.Sprintf("wmctrl -i -r %d -t %d", win, ruleWSIndex), nil, nil, false, false)
				if err != nil {
					l.Warnw("[processWindows]", "wmctrl failed, err:", err)
				}
			}
		}
	}
	return nil
}

func handleWindows() {
	l := logger.Sugar()
	defer wg.Done()
	recv := i3.Subscribe(i3.WindowEventType, i3.WorkspaceEventType)
	defer recv.Close()
	var currentWorkspace string
	i3WorkspaceByDesktop := make(map[string]string)
	index := 1
	for _, w := range workspaces.List() {
		i3WorkspaceByDesktop[w] = fmt.Sprintf("%d: %s", index, w)
		l.Debugw("[prepare]", "index (1-based)", index, "w", w)
		index = index + 1
	}
	for recv.Next() {
		switch ev := recv.Event().(type) {
		case *i3.WindowEvent:
			if ev.Change == "title" {
				r, err := rules.MatchTraits(xserver.WindowTraits{
					Title:    ev.Container.WindowProperties.Title,
					Class:    ev.Container.WindowProperties.Class,
					Instance: ev.Container.WindowProperties.Instance,
					Role:     ev.Container.WindowProperties.Role,
				})
				if err != nil {
					l.Warnw("[handleWindows]", "err", err)
					continue
				}
				l.Debugw("[handleWindows]",
					"event", ev.Change,
					"window", fmt.Sprintf("'%s'/'%s'", ev.Container.WindowProperties.Class, ev.Container.WindowProperties.Title),
					"rule", r,
				)
				if r != nil {
					ruleWorkspace, ok := i3WorkspaceByDesktop[r.Desktop]
					if !ok {
						l.Warnw("[handleWindows]", "unknown rule desktop", r.Desktop)
						continue
					} else {
						l.Debugw("[handleWindows]", "ruleWorkspace", ruleWorkspace, "currentWorkspace", currentWorkspace)
						if ruleWorkspace != currentWorkspace {
							var cmd string
							if r.Activate {
								cmd = fmt.Sprintf("[con_id=\"%v\"] move to workspace %s; workspace %s", ev.Container.ID, ruleWorkspace, ruleWorkspace)
							} else {
								cmd = fmt.Sprintf("[con_id=\"%v\"] move to workspace %s", ev.Container.ID, ruleWorkspace)
							}
							_, err := i3.RunCommand(cmd)
							if err != nil {
								l.Warnw("[handleWindows]", "command failed", cmd, "err", err)
							}
						}
					}
				}
			}
		case *i3.WorkspaceEvent:
			if ev.Change == "focus" {
				l.Debugw("[handleWindows]", "current workspace", currentWorkspace)
				currentWorkspace = ev.Current.Name
			}
		}
	}
}

func perform(ctx *cli.Context) error {
	populateMetadata()
	if ctx.Bool("oneshot") {
		return processWindows()
	} else {
		wg.Add(1)
		go handleWindows()
		wg.Wait()
	}
	return nil
}

func createCLI() *cli.App {
	app := cli.NewApp()
	app.Name = "i3-desktops"
	app.Usage = "Relocates/moves windows according to window mapping rules"
	app.Description = "i3-desktops"
	app.Version = "0.0.1#master"

	app.Flags = []cli.Flag{
		&cli.BoolFlag{
			Name:     "oneshot",
			Usage:    "Iterate and relocate existing windows once",
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
	app := createCLI()
	err := app.Run(os.Args)
	if err != nil {
		l.Errorw("[main]", "err", err)
	}
}
