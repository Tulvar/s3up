package cli

import (
	"context"
	"flag"
	"fmt"
	"io"

	"github.com/tulvar/s3up/pkg/version"
)

type CLI struct {
	stdout io.Writer
	stderr io.Writer
}

func New(stdout, stderr io.Writer) CLI {
	return CLI{stdout: stdout, stderr: stderr}
}

func (c CLI) Run(ctx context.Context, args []string) error {
	if len(args) == 0 {
		c.printUsage()
		return nil
	}

	switch args[0] {
	case "upload", "up":
		return c.runUpload(ctx, args[1:])
	case "sync":
		return c.runSync(ctx, args[1:])
	case "delete", "rm":
		return c.runDelete(ctx, args[1:])
	case "ls", "list":
		return c.runLS(ctx, args[1:])
	case "version":
		fmt.Fprintln(c.stdout, version.Version)
		return nil
	case "help", "-h", "--help":
		c.printUsage()
		return nil
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func (c CLI) printUsage() {
	fmt.Fprintln(c.stdout, "s3up - small S3 upload CLI")
	fmt.Fprintln(c.stdout)
	fmt.Fprintln(c.stdout, "Usage:")
	fmt.Fprintln(c.stdout, "  s3up upload [flags] <local-path> <s3://bucket/key>")
	fmt.Fprintln(c.stdout, "  s3up sync [flags] <local-path> <s3://bucket/prefix>")
	fmt.Fprintln(c.stdout, "  s3up delete [flags] <s3://bucket/key-or-prefix>")
	fmt.Fprintln(c.stdout, "  s3up ls [flags] <s3://bucket/prefix>")
	fmt.Fprintln(c.stdout, "  s3up version")
	fmt.Fprintln(c.stdout)
	fmt.Fprintln(c.stdout, "Upload flags:")
	fs := flag.NewFlagSet("upload", flag.ContinueOnError)
	registerUploadFlags(fs, nil)
	fs.SetOutput(c.stdout)
	fs.PrintDefaults()
	fmt.Fprintln(c.stdout)
	fmt.Fprintln(c.stdout, "LS flags:")
	lsFlags := flag.NewFlagSet("ls", flag.ContinueOnError)
	registerLSFlags(lsFlags, nil)
	lsFlags.SetOutput(c.stdout)
	lsFlags.PrintDefaults()
	fmt.Fprintln(c.stdout)
	fmt.Fprintln(c.stdout, "Sync flags:")
	syncFlags := flag.NewFlagSet("sync", flag.ContinueOnError)
	registerSyncFlags(syncFlags, nil)
	syncFlags.SetOutput(c.stdout)
	syncFlags.PrintDefaults()
	fmt.Fprintln(c.stdout)
	fmt.Fprintln(c.stdout, "Delete flags:")
	deleteFlags := flag.NewFlagSet("delete", flag.ContinueOnError)
	registerDeleteFlags(deleteFlags, nil)
	deleteFlags.SetOutput(c.stdout)
	deleteFlags.PrintDefaults()
}
