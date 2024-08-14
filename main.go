package main

import (
	"context"
	"log"
	"os"
	"os/signal"

	"github.com/go-logr/stdr"
	"github.com/rh-ecosystem-edge/gitstream/cmd/cli"
)

func main() {
	logger := stdr.New(
		log.New(os.Stdout, "", log.Lshortfile),
	)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	app := cli.App{Logger: logger}

	if err := app.GetCLIApp().RunContext(ctx, os.Args); err != nil {
		logger.Error(err, "Application error")
		os.Exit(1)
	}
}
