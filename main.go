package main

import (
	"embed"
	"os"

	"github.com/slinxlink/node/internal/app"
	"github.com/slinxlink/node/internal/bootstrap"
	"github.com/slinxlink/node/internal/cli"
	"github.com/slinxlink/node/internal/server"
	"github.com/slinxlink/node/internal/setup"
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
