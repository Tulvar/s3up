package app

import (
	"context"
	"io"

	"github.com/tulvar/s3up/internal/cli"
)

func Run(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	return cli.New(stdout, stderr).Run(ctx, args)
}
