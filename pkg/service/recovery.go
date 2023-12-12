package service

import (
	"context"
	"fmt"
	"net/http"
	"runtime"
	"runtime/debug"
	"strings"

	"github.com/gorilla/mux"
	"github.com/rs/zerolog"
)

const (
	StackLevelSimple int = iota
	StackLevelFullStack
)

type RecoveryFunc struct {
	StackLevel int
}

func (s *RecoveryFunc) TryToRecover() mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if p := recover(); p != nil {
					w.WriteHeader(http.StatusInternalServerError)
					s.logError(r.Context(), p)
					//TODO: if the header was already written this will log an error
					return
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

func (s *RecoveryFunc) logError(ctx context.Context, a any) {
	logger := zerolog.Ctx(ctx)
	logger.Error().Msgf("Recovering from fatal error: %v", a)
	if s.StackLevel == StackLevelFullStack {
		logger.Error().Msg(string(debug.Stack()))
	} else {
		logger.Error().Msg(format(walk()))
	}
}

func format(f []*runtime.Frame) string {
	var result string
	for _, ff := range f {
		result = fmt.Sprintf("%s\n%v", result, ff)
	}
	return result
}

func walk() []*runtime.Frame {
	var pcs [40]uintptr
	var frames [40]*runtime.Frame
	runtime.Callers(4, pcs[:])
	fs := runtime.CallersFrames(pcs[:])
	more := true
	var f runtime.Frame
	i := 0
	skip := true
	for more && i < 40 {
		f, more = fs.Next()
		if skip {
			if !strings.Contains(f.Function, "panic") {
				skip = false
			}
		} else {
			func1 := f
			frames[i] = &func1
			i++
		}
	}
	ret := frames[:i]
	return ret
}
