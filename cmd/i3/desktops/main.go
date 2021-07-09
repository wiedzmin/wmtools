package main

import (
	"fmt"
	"os"
	"sync"

	"github.com/wiedzmin/toolbox/impl"
	"github.com/wiedzmin/toolbox/impl/xserver"
	"go.i3wm.org/i3"
	"go.uber.org/zap"
)

var (
	logger *zap.Logger
	wg     sync.WaitGroup

	rules           *xserver.WindowRules
	workspaces      *xserver.Workspaces
	i3wsByWorkspace map[string]string

	currentWorkspace string
)

func prepare() {
	l := logger.Sugar()
	var err error
	rules, err = xserver.WindowRulesFromRedis("wm/window_rules")
	if err != nil {
		l.Warnw("[prepare]", "err", err)
		os.Exit(1)
	}
	for _, r := range rules.List() {
		l.Debugw("[prepare]", "r", r)
	}
	workspaces, err = xserver.WorkspacesFromRedis("wm/workspaces")
	if err != nil {
		l.Warnw("[prepare]", "err", err)
		os.Exit(1)
	}
	i3wsByWorkspace = make(map[string]string)
	index := 1
	for _, w := range workspaces.List() {
		i3wsByWorkspace[w] = fmt.Sprintf("%d: %s", index, w)
		l.Debugw("[prepare]", "index", index, "w", w)
		index = index + 1
	}
}

func handleWindows() {
	l := logger.Sugar()
	defer wg.Done()
	recv := i3.Subscribe(i3.WindowEventType, i3.WorkspaceEventType)
	defer recv.Close()
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
					ruleWorkspace, ok := i3wsByWorkspace[r.Desktop]
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

func main() {
	logger = impl.NewLogger()
	defer logger.Sync()
	prepare()
	wg.Add(1)
	go handleWindows()
	wg.Wait()
}
