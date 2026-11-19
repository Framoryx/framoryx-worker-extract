package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Framoryx/framoryx-worker-extract/internal/platform/logging"
	"github.com/Framoryx/framoryx-worker-extract/internal/platform/telemetry"
)

func main() {
	slog.SetDefault(logging.NewJSON(os.Stdout, telemetry.ServiceName, nil))
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := run(ctx); err != nil {
		slog.Error("Falha no worker", "error", err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	tel, err := telemetry.New(ctx)
	if err != nil {
		return err
	}
	slog.InfoContext(ctx, "Rastreabilidade inicializada; consumidor de mensagens ainda não configurado")
	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return tel.Shutdown(shutdownCtx)
}
