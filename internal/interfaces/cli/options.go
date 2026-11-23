package cli

import (
	"flag"
	"fmt"
	"strings"
)

// parseOptions enforces short and long option spelling and stops at the first positional argument.
func parseOptions(fs *flag.FlagSet, args []string) error {
	for len(args) > 0 {
		arg := args[0]
		if arg == "-" || arg == "--" || !strings.HasPrefix(arg, "-") {
			break
		}
		args = args[1:]
		option, value, inline := strings.Cut(arg, "=")
		name := strings.TrimLeft(option, "-")
		prefix := "--"
		if len(name) == 1 {
			prefix = "-"
		}
		if name == "" {
			return fmt.Errorf("invalid option %q", option)
		}
		if option != prefix+name {
			return fmt.Errorf("invalid option %q; use %s%s", option, prefix, name)
		}
		if name == "help" || name == "h" {
			fs.Usage()
			return flag.ErrHelp
		}
		f := fs.Lookup(name)
		if f == nil {
			return fmt.Errorf("unknown option %s", option)
		}
		if !inline {
			if b, ok := f.Value.(interface{ IsBoolFlag() bool }); ok && b.IsBoolFlag() {
				value = "true"
			} else if len(args) > 0 {
				value, args = args[0], args[1:]
			} else {
				return fmt.Errorf("%s requires a value", option)
			}
		}
		if err := fs.Set(name, value); err != nil {
			return fmt.Errorf("%s: %w", option, err)
		}
	}
	return fs.Parse(args)
}
