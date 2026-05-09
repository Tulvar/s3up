package cli

import (
	"flag"
	"strings"
)

type boolFlag interface {
	IsBoolFlag() bool
}

func parseFlags(fs *flag.FlagSet, args []string) error {
	return fs.Parse(reorderFlags(fs, args))
}

func reorderFlags(fs *flag.FlagSet, args []string) []string {
	flags := make([]string, 0, len(args))
	positionals := make([]string, 0, len(args))

	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			positionals = append(positionals, args[i+1:]...)
			break
		}
		if !isFlagArg(arg) {
			positionals = append(positionals, arg)
			continue
		}

		flags = append(flags, arg)
		name, _, hasValue := splitFlagArg(arg)
		flagDef := fs.Lookup(name)
		if flagDef == nil || hasValue || isBoolFlag(flagDef) {
			continue
		}
		if i+1 < len(args) {
			i++
			flags = append(flags, args[i])
		}
	}

	return append(flags, positionals...)
}

func isFlagArg(arg string) bool {
	return strings.HasPrefix(arg, "-") && arg != "-"
}

func splitFlagArg(arg string) (name, value string, hasValue bool) {
	trimmed := strings.TrimLeft(arg, "-")
	name, value, hasValue = strings.Cut(trimmed, "=")
	return name, value, hasValue
}

func isBoolFlag(flagDef *flag.Flag) bool {
	value, ok := flagDef.Value.(boolFlag)
	return ok && value.IsBoolFlag()
}
