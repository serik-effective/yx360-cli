package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"github.com/effective-dev-os/yx360-cli/internal/cli"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	if err := cli.NewRootCmd().ExecuteContext(ctx); err != nil {
		fmt.Fprintln(os.Stderr, "yx360:", err)
		os.Exit(1)
	}
}
