package main

import (
	"embed"
	"os"

	"github.com/gzjacktang/single-ui/internal/app"
	"github.com/gzjacktang/single-ui/internal/bootstrap"
	"github.com/gzjacktang/single-ui/internal/cli"
	"github.com/gzjacktang/single-ui/internal/server"
	"github.com/gzjacktang/single-ui/internal/setup"
)

var Version = "dev"

//go:embed web/dist
var webFS embed.FS

func main() {
	if len(os.Args) > 1 && os.Args[1] == "cli" {
		cli.Start(Version)
		return
	}
	if len(os.Args) > 1 && os.Args[1] == "setup" {
		setup.ExitOnError(setup.Command(os.Args[2:]))
		return
	}
	app.Version = Version
	server.Init(webFS)
	bootstrap.Start()
}
