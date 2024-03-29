package stream

import (
	"context"

	"github.com/andrescosta/goico/pkg/broadcaster"
	"github.com/rs/zerolog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

func Recv[T proto.Message](ctx context.Context, s grpc.ClientStream, bc *broadcaster.Broadcaster[T]) error {
	logger := zerolog.Ctx(ctx)
	for {
		select {
		case <-ctx.Done():
			logger.Debug().Msg("End signal recv")
			if err := s.CloseSend(); err != nil {
				logger.Warn().AnErr("error", err).Msg("Listen: Error while closing stream.")
			}
			return ctx.Err()
		case <-s.Context().Done():
			logger.Debug().Msg("End stream signal recv")
			if err := s.CloseSend(); err != nil {
				logger.Warn().AnErr("error", err).Msg("Listen: Error while closing stream.")
			}
			return s.Context().Err()
		default:
			var t T
			p := t.ProtoReflect().New()
			err := s.RecvMsg(p.Interface())
			if err != nil {
				if status.Code(err) != codes.Canceled {
					logger.Warn().AnErr("error", err).Msg("Listen: error getting message")
				}
				continue
			}
			if err := bc.Write(p.Interface().(T)); err != nil {
				return err
			}
		}
	}
}
