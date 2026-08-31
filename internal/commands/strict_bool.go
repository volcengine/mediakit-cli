package commands

import (
	"fmt"

	"github.com/spf13/pflag"
)

type strictBoolValue struct {
	target *bool
}

func (value *strictBoolValue) Set(raw string) error {
	switch raw {
	case "true":
		*value.target = true
		return nil
	case "false":
		*value.target = false
		return nil
	default:
		return fmt.Errorf("must be true or false")
	}
}

func (value *strictBoolValue) String() string {
	if value == nil || value.target == nil || !*value.target {
		return "false"
	}
	return "true"
}

func (value *strictBoolValue) Type() string {
	return "bool"
}

func (value *strictBoolValue) Get() any {
	return *value.target
}

func strictBool(flags *pflag.FlagSet, name string, defaultValue bool, usage string) {
	var target bool
	strictBoolVar(flags, &target, name, defaultValue, usage)
}

func strictBoolVar(
	flags *pflag.FlagSet,
	target *bool,
	name string,
	defaultValue bool,
	usage string,
) {
	*target = defaultValue
	flags.Var(&strictBoolValue{target: target}, name, usage)
	flag := flags.Lookup(name)
	flag.DefValue = (&strictBoolValue{target: target}).String()
	flag.NoOptDefVal = "true"
}
