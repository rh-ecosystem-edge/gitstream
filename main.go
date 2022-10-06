package main

import (
	"context"
	"log"
	"os"
	"os/signal"

	"github.com/go-logr/stdr"
	"github.com/qbarrand/gitstream/cmd/cli"
	"github.com/qbarrand/gitstream/internal/intents"
	"github.com/qbarrand/gitstream/internal/markup"
)

func main() {
	logger := stdr.New(
		log.New(os.Stdout, "", log.Lshortfile),
	)

	finder, err := markup.NewFinder("Upstream-Commit")
	if err != nil {
		logger.Error(err, "Could not create a new markup finder")
		os.Exit(1)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	app := cli.App{
		IntentsGetter: intents.NewIntentsGetter(finder, logger.V(1)),
		Logger:        logger,
	}

	if err := app.GetCLIApp().RunContext(ctx, os.Args); err != nil {
		logger.Error(err, "Application error")
		os.Exit(1)
	}
}
