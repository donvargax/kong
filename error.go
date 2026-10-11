package kong

import (
	"fmt"
	"strconv"
	"strings"
)

// ParseError is the error type returned by Kong.Parse().
//
// It contains the parse Context that triggered the error.
//
// When the command-line itself is at fault, the error it wraps is one of the
// error types below, which can be retrieved with errors.As.
type ParseError struct {
	error
	Context  *Context
	exitCode int
}

// Unwrap returns the original cause of the error.
func (p *ParseError) Unwrap() error { return p.error }

// ExitCode returns the status that Kong should exit with if it fails with a ParseError.
func (p *ParseError) ExitCode() int {
	if p.exitCode == 0 {
		return exitNotOk
	}
	return p.exitCode
}

// UnknownFlagError is returned when the command-line contains a flag that is
// not defined.
type UnknownFlagError struct {
	// Flag as it was written on the command-line, eg. "--nam" or "-x".
	Flag string
	// Candidates are the defined flags close to Flag, as suggested in the message.
	Candidates []string
}

func (e *UnknownFlagError) Error() string {
	return withCandidates("unknown flag "+e.Flag, e.Candidates)
}

// UnexpectedArgumentError is returned when the command-line contains an
// argument that matches no command or positional argument.
type UnexpectedArgumentError struct {
	// Arg as it was written on the command-line.
	Arg string
	// ExpectedCommand is true if a command was expected in place of Arg, ie. Arg
	// is an unknown command rather than an extra positional argument.
	ExpectedCommand bool
	// Candidates are the commands close to Arg, as suggested in the message.
	Candidates []string
}

func (e *UnexpectedArgumentError) Error() string {
	return withCandidates("unexpected argument "+e.Arg, e.Candidates)
}

// UnexpectedTokenError is returned when the command-line contains a token
// that is not valid where it occurs, such as a value attached to a flag that
// does not take one.
type UnexpectedTokenError struct {
	Token Token
}

func (e *UnexpectedTokenError) Error() string {
	if e.Token.Type == FlagValueToken {
		return fmt.Sprintf("unexpected flag argument %q", e.Token.Value)
	}
	return fmt.Sprintf("unexpected token %s", e.Token)
}

// MissingValueError is returned when a value is expected but the command-line
// ends or continues with a flag instead.
type MissingValueError struct {
	// Value is the flag or positional argument missing a value, or nil if the
	// error did not occur while parsing one.
	Value *Value
	// Expected describes the type of value expected, eg. "string" or "int".
	Expected string
	// Token found in place of the value.
	Token Token
}

func (e *MissingValueError) Error() string {
	msg := fmt.Sprintf("expected %s value but got %q (%s)", e.Expected, e.Token, e.Token.InferredType())
	if e.Value != nil {
		return e.Value.ShortSummary() + ": " + msg
	}
	return msg
}

// InvalidValueError is returned when the Mapper of a flag or positional
// argument fails to decode its value.
type InvalidValueError struct {
	// Value is the flag or positional argument.
	Value *Value
	// Err is the error returned by the Mapper. A Mapper also fails this way
	// when applied to a field of a type it does not support.
	Err error
}

func (e *InvalidValueError) Error() string { return e.Value.ShortSummary() + ": " + e.Err.Error() }

// Unwrap returns the error returned by the Mapper.
func (e *InvalidValueError) Unwrap() error { return e.Err }

// EnumError is returned when the value of a flag or positional argument is
// not one of its enum values.
type EnumError struct {
	// Value is the flag or positional argument.
	Value *Value
	// Got is the value given.
	Got string
	// Allowed are the enum values.
	Allowed []string
}

func (e *EnumError) Error() string {
	return fmt.Sprintf("%s must be one of %s but got %q", e.Value.ShortSummary(), quoteAll(e.Allowed, ","), e.Got)
}

// MissingFlagsError is returned when required flags are missing.
type MissingFlagsError struct {
	// Flags that are missing, in the order the message names them. A flag of an
	// xor group is only missing if no flag of its group was given.
	Flags []*Flag
	// missing holds each flag or group of flags as the message names it.
	missing []string
}

func (e *MissingFlagsError) Error() string {
	missing := e.missing
	if len(missing) == 0 {
		for _, flag := range e.Flags {
			missing = append(missing, flag.Summary())
		}
	}
	return "missing flags: " + strings.Join(missing, ", ")
}

// MissingPositionalsError is returned when required positional arguments are
// missing.
type MissingPositionalsError struct {
	Positionals []*Positional
	// expected selects the message used when none of the positional arguments were given.
	expected bool
}

func (e *MissingPositionalsError) Error() string {
	if e.expected {
		return "expected " + quoteSummaries(e.Positionals)
	}
	missing := make([]string, 0, len(e.Positionals))
	for _, arg := range e.Positionals {
		missing = append(missing, "<"+arg.Name+">")
	}
	return "missing positional arguments " + strings.Join(missing, " ")
}

func quoteSummaries(positionals []*Positional) string {
	summaries := make([]string, 0, len(positionals))
	for _, arg := range positionals {
		summaries = append(summaries, arg.Summary())
	}
	return strconv.Quote(strings.Join(summaries, " "))
}

// MissingCommandError is returned when a command or branching positional
// argument is required but missing.
type MissingCommandError struct {
	// Candidates are the commands and branching positional arguments, one of which is required.
	Candidates []*Node
	// positionals that are also missing, when commands were added to an
	// application with positional arguments by Embed() or DynamicCommand().
	positionals []*Positional
}

func (e *MissingCommandError) Error() string {
	missing := make([]string, 0, len(e.Candidates)+1)
	if len(e.positionals) > 0 {
		missing = append(missing, quoteSummaries(e.positionals))
	}
	for _, node := range e.Candidates {
		if node.Type == ArgumentNode {
			missing = append(missing, strconv.Quote(node.Summary()))
		} else {
			missing = append(missing, strconv.Quote(node.Name))
		}
	}
	if len(missing) > 5 {
		missing = append(missing[:5], "...")
	}
	if len(missing) == 1 {
		return "expected " + missing[0]
	}
	return "expected one of " + strings.Join(missing, ", ")
}

// XorFlagsError is returned when more than one flag of an xor group is given.
type XorFlagsError struct {
	// Group is the name of the xor group.
	Group string
	// Flags given together.
	Flags []*Flag
}

func (e *XorFlagsError) Error() string { return joinFlagNames(e.Flags) + " can't be used together" }

// AndFlagsError is returned when some but not all flags of an and group are
// given.
type AndFlagsError struct {
	// Group is the name of the and group.
	Group string
	// Flags of the group.
	Flags []*Flag
}

func (e *AndFlagsError) Error() string { return joinFlagNames(e.Flags) + " must be used together" }

// parseErrors are reported together, separated by commas.
type parseErrors []error

func (e parseErrors) Error() string {
	msgs := make([]string, 0, len(e))
	for _, err := range e {
		msgs = append(msgs, err.Error())
	}
	return strings.Join(msgs, ", ")
}

func (e parseErrors) Unwrap() []error { return e }

// joinErrors returns nil, the only error, or the errors reported together.
func joinErrors(errs []error) error {
	switch len(errs) {
	case 0:
		return nil
	case 1:
		return errs[0]
	default:
		return parseErrors(errs)
	}
}

func joinFlagNames(flags []*Flag) string {
	names := make([]string, 0, len(flags))
	for _, flag := range flags {
		names = append(names, "--"+flag.Name)
	}
	return strings.Join(names, " and ")
}

func quoteAll(values []string, sep string) string {
	quoted := make([]string, 0, len(values))
	for _, value := range values {
		quoted = append(quoted, strconv.Quote(value))
	}
	return strings.Join(quoted, sep)
}

// closestCandidates returns the candidates close to needle.
func closestCandidates(needle string, haystack []string) []string {
	var closest []string
	for _, candidate := range haystack {
		if strings.HasPrefix(candidate, needle) || levenshtein(candidate, needle) <= 2 {
			closest = append(closest, candidate)
		}
	}
	return closest
}

// withCandidates suggests the candidates after msg.
func withCandidates(msg string, candidates []string) string {
	switch len(candidates) {
	case 0:
		return msg
	case 1:
		return fmt.Sprintf("%s, did you mean %q?", msg, candidates[0])
	default:
		return fmt.Sprintf("%s, did you mean one of %s?", msg, quoteAll(candidates, ", "))
	}
}
