package cli

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"syscall"
)

func Execute(ctx context.Context) error {
	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	defer cancel()

	err := cmd.ExecuteContext(ctx)
	if err != nil && errors.Is(err, context.Canceled) {
		return nil
	}

	return err
}
