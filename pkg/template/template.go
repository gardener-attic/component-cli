// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package template

import (
	"github.com/drone/envsubst"
)

// Options defines the options for component-cli templating
type Options struct {
	Vars map[string]string
}

// Usage prints out the usage for templating
func (o *Options) Usage() string {
	return `
Templating:
All yaml/json defined resources can be templated using simple envsubst syntax.
Variables are specified after a "--" and follow the syntax "<name>=<value>".

Note: Variable names are case-sensitive.

Example:
<pre>
<command> [args] [--flags] -- MY_VAL=test
</pre>

<pre>

key:
  subkey: "abc ${MY_VAL}"

</pre>

`
}

// Complete parses commandline argument variables.
func (o *Options) Complete(args []string) error {
	parseVars := false
	o.Vars = make(map[string]string)
	for _, arg := range args {
		// only start parsing after "--"
		if arg == "--" {
			parseVars = true
			continue
		}
		if !parseVars {
			continue
		}

		// parse value
		name := ""
		value := ""
		for i := 1; i < len(arg); i++ { // equals cannot be first
			if arg[i] == '=' {
				value = arg[i+1:]
				name = arg[0:i]
				break
			}
		}
		o.Vars[name] = value
	}
	return nil
}

// Template templates a string with the parsed vars.
func (o *Options) Template(data string) (string, error) {
	return envsubst.Eval(data, o.mapping)
}

// mapping is a helper function for the envsubst to provide the value for a variable name.
// It returns an emtpy string if the variable is not defined.
func (o *Options) mapping(variable string) string {
	if o.Vars == nil {
		return ""
	}
	// todo: maybe use os.getenv as backup.
	return o.Vars[variable]
}
