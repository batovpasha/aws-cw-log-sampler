package cli

import (
	"errors"
	"flag"

	"github.com/batovpasha/aws-cw-log-sampler/internal/sample"
)

type CommonFlags struct {
	LogGroupNamePattern  string
	DstGroup             string
	Type                 string
	RandLogStreamsNumber int
}

func RegisterCommonFlags(fs *flag.FlagSet) *CommonFlags {
	f := &CommonFlags{}
	fs.StringVar(&f.LogGroupNamePattern, "logGroupNamePattern", "", "log group name pattern")
	fs.StringVar(&f.DstGroup, "dstGroup", "", "destination log group")
	fs.StringVar(&f.Type, "type", "", "sampling type")
	fs.IntVar(&f.RandLogStreamsNumber, "randLogStreamsNumber", 0, "number of random log streams")
	return f
}

func ValidateCommonFlags(flags *CommonFlags) error {
	var errs []error

	if flags.LogGroupNamePattern == "" {
		errs = append(errs, errors.New("--logGroupNamePattern is required"))
	}
	if flags.DstGroup == "" {
		errs = append(errs, errors.New("--dstGroup is required"))
	}

	switch flags.Type {
	case sample.TypeRandLogStreams:
		if flags.RandLogStreamsNumber < 1 || flags.RandLogStreamsNumber > 100 {
			errs = append(
				errs,
				errors.New("--randLogStreamsNumber should be an integer less than 100"),
			)
		}
	case "":
		errs = append(errs, errors.New("--type is required"))
	default:
		errs = append(errs, errors.New("--type has invalid value"))
	}

	return errors.Join(errs...)
}
