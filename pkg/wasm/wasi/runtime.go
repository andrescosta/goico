package wasi

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/pkg/errors"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
	"github.com/tetratelabs/wazero/sys"
)

type Runtime struct{}

// experimental: not used
func New(ctx context.Context) (*Runtime, error) {
	r := wazero.NewRuntime(ctx)

	defer r.Close(ctx)

	config := wazero.NewModuleConfig().
		WithStdout(os.Stdout).WithStderr(os.Stderr)

	wasi_snapshot_preview1.MustInstantiate(ctx, r)

	var catWasm []byte

	if _, err := r.InstantiateWithConfig(ctx, catWasm, config.WithArgs("wasi", os.Args[1])); err != nil {
		exitErr := &sys.ExitError{}
		if ok := errors.As(err, &exitErr); ok && exitErr.ExitCode() != 0 {
			fmt.Fprintf(os.Stderr, "exit_code: %d\n", exitErr.ExitCode())
		} else if !ok {
			log.Panicln(err)
		}
	}

	return nil, nil
}
