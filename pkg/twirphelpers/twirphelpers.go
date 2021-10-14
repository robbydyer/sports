package twirphelpers

import (
	"context"

	"github.com/twitchtv/twirp"
	"go.uber.org/zap"

	"github.com/robbydyer/sports/pkg/board"
)

// GetDefaultHooks returns default custom twirp.ServerHooks
func GetDefaultHooks(board board.Board, logger *zap.Logger) *twirp.ServerHooks {
	boardName := ""
	if board != nil {
		boardName = board.Name()
	}
	return &twirp.ServerHooks{
		RequestRouted: func(ctx context.Context) (context.Context, error) {
			method, _ := twirp.MethodName(ctx)
			pkg, _ := twirp.PackageName(ctx)
			svc, _ := twirp.ServiceName(ctx)
			logger.Info("twirp request routed",
				zap.String("method", method),
				zap.String("service", svc),
				zap.String("package", pkg),
				zap.String("board", boardName),
			)

			return ctx, nil
		},
		Error: func(ctx context.Context, err twirp.Error) context.Context {
			pkg, _ := twirp.PackageName(ctx)
			svc, _ := twirp.ServiceName(ctx)
			logger.Error("twirp API error",
				zap.Error(err),
				zap.String("service", svc),
				zap.String("package", pkg),
				zap.String("board", boardName),
			)

			return ctx
		},
	}
}
