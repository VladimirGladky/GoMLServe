package main

import (
	"GoMLServe/internal/app"
	"GoMLServe/internal/config"
	"GoMLServe/pkg/logger"
	"context"
	"fmt"
	"go.uber.org/zap"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cfg, err := config.New()
	fmt.Println(cfg)
	if err != nil {
		logger.GetLoggerFromCtx(ctx).Error("error loading config: %v", zap.Error(err))
		return
	}
	ctx, _ = logger.New(ctx)
	application := app.New(ctx, cfg)
	application.MustRun()
}
