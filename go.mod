module github.com/wiedzmin/i3tools

go 1.16

require (
	github.com/BurntSushi/xgb v0.0.0-20210121224620-deaf085860bc // indirect
	github.com/BurntSushi/xgbutil v0.0.0-20190907113008-ad855c713046 // indirect
	github.com/urfave/cli/v2 v2.3.0
	github.com/wiedzmin/toolbox v0.0.0-20210913182605-2a753d3299da
	go.i3wm.org/i3 v0.0.0-20190720062127-36e6ec85cc5a
	go.uber.org/zap v1.17.0
)

// replace (
//     github.com/wiedzmin/toolbox => ../toolbox
//     go.i3wm.org/i3 => ../../i3/go-i3
// )
