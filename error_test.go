package kong_test

import (
	"errors"
	"testing"

	"github.com/alecthomas/assert/v2"

	"github.com/donvargax/kong"
)

// parseErrorAs parses args and returns the error it fails with, which must be a
// ParseError wrapping an E.
func parseErrorAs[E error](t *testing.T, cli any, args ...string) (E, error) {
	t.Helper()
	_, err := mustNew(t, cli).Parse(args)
	var parseErr *kong.ParseError
	assert.True(t, errors.As(err, &parseErr), "%v", err)
	var target E
	assert.True(t, errors.As(err, &target), "%T: %v", parseErr.Unwrap(), err)
	return target, err
}

func flagNames(flags []*kong.Flag) []string {
	names := make([]string, 0, len(flags))
	for _, flag := range flags {
		names = append(names, flag.Name)
	}
	return names
}

func TestUnknownFlagError(t *testing.T) {
	var cli struct {
		Name string `short:"n"`
	}
	unknown, err := parseErrorAs[*kong.UnknownFlagError](t, &cli, "--nam")
	assert.EqualError(t, err, `unknown flag --nam, did you mean "--name"?`)
	assert.Equal(t, "--nam", unknown.Flag)
	assert.Equal(t, []string{"--name"}, unknown.Candidates)

	unknown, err = parseErrorAs[*kong.UnknownFlagError](t, &cli, "-x")
	assert.EqualError(t, err, `unknown flag -x, did you mean one of "-h", "-n"?`)
	assert.Equal(t, "-x", unknown.Flag)
	assert.Equal(t, []string{"-h", "-n"}, unknown.Candidates)

	unknown, err = parseErrorAs[*kong.UnknownFlagError](t, &cli, "--verbose")
	assert.EqualError(t, err, "unknown flag --verbose")
	assert.Equal(t, "--verbose", unknown.Flag)
	assert.Zero(t, unknown.Candidates)
}

func TestUnexpectedArgumentError(t *testing.T) {
	var commands struct {
		Build struct{} `cmd:""`
		Bench struct{} `cmd:""`
	}
	unexpected, err := parseErrorAs[*kong.UnexpectedArgumentError](t, &commands, "bulid")
	assert.EqualError(t, err, `unexpected argument bulid, did you mean "build"?`)
	assert.Equal(t, "bulid", unexpected.Arg)
	assert.True(t, unexpected.ExpectedCommand)
	assert.Equal(t, []string{"build"}, unexpected.Candidates)

	var positionals struct {
		Path string `arg:""`
	}
	unexpected, err = parseErrorAs[*kong.UnexpectedArgumentError](t, &positionals, "one", "two")
	assert.EqualError(t, err, "unexpected argument two")
	assert.Equal(t, "two", unexpected.Arg)
	assert.False(t, unexpected.ExpectedCommand)
	assert.Zero(t, unexpected.Candidates)
}

// valueless is a flag that takes no value.
type valueless bool

func (v *valueless) Decode(ctx *kong.DecodeContext) error {
	*v = true
	return nil
}

func TestUnexpectedTokenError(t *testing.T) {
	var cli struct {
		Flag valueless
	}
	unexpected, err := parseErrorAs[*kong.UnexpectedTokenError](t, &cli, "--flag=value")
	assert.EqualError(t, err, `unexpected flag argument "value"`)
	assert.Equal(t, kong.Token{Type: kong.FlagValueToken, Value: "value"}, unexpected.Token)
}

func TestMissingValueError(t *testing.T) {
	var cli struct {
		Name  string
		Other bool
	}
	missing, err := parseErrorAs[*kong.MissingValueError](t, &cli, "--name")
	assert.EqualError(t, err, `--name: expected string value but got "EOL" (<EOL>)`)
	assert.Equal(t, "name", missing.Value.Name)
	assert.Equal(t, "string", missing.Expected)
	assert.Equal(t, kong.EOLToken, missing.Token.Type)
	var invalid *kong.InvalidValueError
	assert.False(t, errors.As(err, &invalid))

	missing, err = parseErrorAs[*kong.MissingValueError](t, &cli, "--name", "--other")
	assert.EqualError(t, err, `--name: expected string value but got "--other" (long flag); perhaps try --name="--other"?`)
	assert.Equal(t, "name", missing.Value.Name)
	assert.Equal(t, "--other", missing.Token.String())
}

func TestInvalidValueError(t *testing.T) {
	var cli struct {
		Count int
		Sizes []int `arg:"" optional:""`
	}
	invalid, err := parseErrorAs[*kong.InvalidValueError](t, &cli, "--count=many")
	assert.EqualError(t, err, `--count: expected a valid 64 bit int but got "many"`)
	assert.Equal(t, "count", invalid.Value.Name)
	assert.EqualError(t, invalid.Err, `expected a valid 64 bit int but got "many"`)

	invalid, err = parseErrorAs[*kong.InvalidValueError](t, &cli, "1", "big")
	assert.EqualError(t, err, `[<sizes> ...]: expected a valid 64 bit int but got "big"`)
	assert.Equal(t, "sizes", invalid.Value.Name)
	assert.True(t, invalid.Value.Flag == nil)
}

func TestInvalidValueErrorFromEnvar(t *testing.T) {
	t.Setenv("KONG_COUNT", "many")
	var cli struct {
		Count int `env:"KONG_COUNT"`
	}
	invalid, err := parseErrorAs[*kong.InvalidValueError](t, &cli)
	assert.EqualError(t, err, `--count: expected a valid 64 bit int but got "many" (from envar KONG_COUNT="many")`)
	assert.Equal(t, "count", invalid.Value.Name)
}

func TestEnumError(t *testing.T) {
	var cli struct {
		Color string `enum:"red,green" default:"red"`
	}
	enum, err := parseErrorAs[*kong.EnumError](t, &cli, "--color=blue")
	assert.EqualError(t, err, `--color must be one of "red","green" but got "blue"`)
	assert.Equal(t, "color", enum.Value.Name)
	assert.Equal(t, "blue", enum.Got)
	assert.Equal(t, []string{"red", "green"}, enum.Allowed)
}

func TestMissingFlagsError(t *testing.T) {
	var cli struct {
		One   string `required:""`
		Two   string `required:"" xor:"choice"`
		Three string `required:"" xor:"choice"`
	}
	missing, err := parseErrorAs[*kong.MissingFlagsError](t, &cli)
	assert.EqualError(t, err, "missing flags: --one=STRING, --two=STRING or --three=STRING")
	assert.Equal(t, []string{"one", "two", "three"}, flagNames(missing.Flags))
}

func TestMissingPositionalsError(t *testing.T) {
	var cli struct {
		One string `arg:""`
		Two string `arg:""`
	}
	missing, err := parseErrorAs[*kong.MissingPositionalsError](t, &cli)
	assert.EqualError(t, err, `expected "<one> <two>"`)
	assert.Equal(t, 2, len(missing.Positionals))
	assert.Equal(t, "one", missing.Positionals[0].Name)
	assert.Equal(t, "two", missing.Positionals[1].Name)

	// Positional arguments missing after some were given.
	missing = &kong.MissingPositionalsError{Positionals: missing.Positionals[1:]}
	assert.EqualError(t, missing, "missing positional arguments <two>")
}

func TestMissingCommandError(t *testing.T) {
	var cli struct {
		Build struct{} `cmd:""`
		Bench struct{} `cmd:""`
	}
	missing, err := parseErrorAs[*kong.MissingCommandError](t, &cli)
	assert.EqualError(t, err, `expected one of "build", "bench"`)
	assert.Equal(t, 2, len(missing.Candidates))
	assert.Equal(t, "build", missing.Candidates[0].Name)
	assert.Equal(t, "bench", missing.Candidates[1].Name)
}

func TestMissingCommandErrorWithPositionals(t *testing.T) {
	var cli struct {
		Path string `arg:""`
	}
	_, err := mustNew(t, &cli, kong.DynamicCommand("build", "", "", &struct{}{})).Parse(nil)
	assert.EqualError(t, err, `expected one of "<path>", "build"`)
	var missing *kong.MissingCommandError
	assert.True(t, errors.As(err, &missing))
	assert.Equal(t, 1, len(missing.Candidates))
	assert.Equal(t, "build", missing.Candidates[0].Name)
}

func TestXorFlagsError(t *testing.T) {
	var cli struct {
		One bool `xor:"choice"`
		Two bool `xor:"choice"`
	}
	xor, err := parseErrorAs[*kong.XorFlagsError](t, &cli, "--one", "--two")
	assert.EqualError(t, err, "--one and --two can't be used together")
	assert.Equal(t, "choice", xor.Group)
	assert.Equal(t, []string{"one", "two"}, flagNames(xor.Flags))
}

func TestAndFlagsError(t *testing.T) {
	var cli struct {
		One bool `and:"both"`
		Two bool `and:"both"`
	}
	and, err := parseErrorAs[*kong.AndFlagsError](t, &cli, "--one")
	assert.EqualError(t, err, "--one and --two must be used together")
	assert.Equal(t, "both", and.Group)
	assert.Equal(t, []string{"one", "two"}, flagNames(and.Flags))
}

func TestXorAndAndFlagsErrorsTogether(t *testing.T) {
	var cli struct {
		One   bool `xor:"choice"`
		Two   bool `xor:"choice"`
		Three bool `and:"both"`
		Four  bool `and:"both"`
	}
	xor, err := parseErrorAs[*kong.XorFlagsError](t, &cli, "--one", "--two", "--three")
	assert.EqualError(t, err, "--one and --two can't be used together, --three and --four must be used together")
	assert.Equal(t, []string{"one", "two"}, flagNames(xor.Flags))
	and, _ := parseErrorAs[*kong.AndFlagsError](t, &cli, "--one", "--two", "--three")
	assert.Equal(t, []string{"three", "four"}, flagNames(and.Flags))
}
