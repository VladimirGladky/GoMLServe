package app

import (
	"GoMLServe/internal/config"
	"GoMLServe/internal/repository"
	"GoMLServe/internal/service"
	"GoMLServe/internal/transport/http"
	"GoMLServe/pkg/logger"
	"GoMLServe/pkg/postgres"
	"context"
	"fmt"
	"go.uber.org/zap"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

type App struct {
	MLServer *http.MLServer
	ctx      context.Context
	cancel   context.CancelFunc
	wg       sync.WaitGroup
}

func New(ctx context.Context, cfg *config.Config) *App {
	db, err := postgres.New(cfg.Postgres)
	if err != nil {
		panic(err)
	}
	mlRepository := repository.NewMLRepository(ctx, db)
	mlService := service.New(ctx, mlRepository)
	mlServer := http.New(cfg, ctx, mlService)
	return &App{
		MLServer: mlServer,
		ctx:      ctx,
	}
}

func (a *App) MustRun() {
	if err := a.Run(); err != nil {
		panic(err)
	}
}

func (a *App) Run() error {
	errCh := make(chan error, 2)
	a.wg.Add(1)
	go func() {
		logger.GetLoggerFromCtx(a.ctx).Info("Server started")
		defer a.wg.Done()
		if err := a.MLServer.Run(); err != nil {
			errCh <- err
			a.cancel()
		}
	}()
	a.wg.Add(1)
	go func() {
		defer a.wg.Done()
		logger.GetLoggerFromCtx(a.ctx).Info("RabbitMQ consumer starting")
		if err := a.MLServer.MLService.StartMLTaskConsumer(); err != nil {
			errCh <- fmt.Errorf("RabbitMQ consumer error: %w", err)
			a.cancel()
		}
	}()
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	select {
	case err := <-errCh:
		logger.GetLoggerFromCtx(a.ctx).Error("error running app", zap.Error(err))
		return err
	case sig := <-sigCh:
		logger.GetLoggerFromCtx(a.ctx).Info("received signal", zap.String("signal", sig.String()))
		a.Stop()
	case <-a.ctx.Done():
		logger.GetLoggerFromCtx(a.ctx).Info("context done")
	}
	return nil
}

func (a *App) Stop() {
	a.cancel()
	a.wg.Wait()
}
