package main

import (
	"bytes"
	"errors"
	"flag"
	"io"
)

const (
	exitSuccess = 0
	exitRuntime = 1
	exitUsage   = 2
)

// newCommandFlagSet keeps subcommands on the same non-terminating flag policy.
// parseCommandFlags assigns the final output stream once parsing determines
// whether the input is help or a usage error.
func newCommandFlagSet(name string) *flag.FlagSet {
	return flag.NewFlagSet(name, flag.ContinueOnError)
}

// parseCommandFlags returns (exit code, true) when parsing fully handles the
// command through either help or a usage error. A false done value means the
// caller may continue executing the command.
func parseCommandFlags(fs *flag.FlagSet, args []string, out, errOut io.Writer) (code int, done bool) {
	var diagnostics bytes.Buffer
	fs.SetOutput(&diagnostics)
	err := fs.Parse(args)
	switch {
	case err == nil:
		fs.SetOutput(errOut)
		return exitSuccess, false
	case errors.Is(err, flag.ErrHelp):
		fs.SetOutput(out)
		fs.Usage()
		fs.SetOutput(errOut)
		return exitSuccess, true
	default:
		_, _ = io.Copy(errOut, &diagnostics)
		fs.SetOutput(errOut)
		return exitUsage, true
	}
}
