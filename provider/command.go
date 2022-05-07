package provider

import (
	"os"
	"os/exec"
	"strings"

	"github.com/pkg/errors"
)

type (
	Cmd struct {
		*exec.Cmd
		Env Environment
	}
	Command interface {
		Command() (string, []string, []CommandOption)
		Execute(result CommandResult) error
		Close() error
	}
	CommandOption func(*Cmd)
	CommandResult = interface{}

	StringCommand struct {
		Cmd       string
		Arguments []string
		Options   []CommandOption
	}

	RemoteCommand struct {
		Cmd Command
		Ssh *Ssh
	}

	Environment map[string][]string
)

func (c *StringCommand) Command() (string, []string, []CommandOption) {
	return c.Cmd, c.Arguments, c.Options
}

func (c *StringCommand) Execute(v interface{}) error {
	command, arguments, options := c.Command()
	return CommandExecuteUnmarshal(command, arguments, nil, v, options...)
}

func (c *StringCommand) Close() error { return nil }

func CommandFromString(command string, arguments ...string) *StringCommand {
	return &StringCommand{
		Cmd:       command,
		Arguments: arguments,
	}
}

//

func (c *RemoteCommand) Command() (string, []string, []CommandOption) {
	return c.Cmd.Command()
}

func (c *RemoteCommand) Execute(v interface{}) error {
	command, arguments, options := c.Cmd.Command()
	sshCommand, sshArguments, sshOptions := c.Ssh.Command()
	return CommandExecuteUnmarshal(
		sshCommand,
		append(append(sshArguments, command), arguments...),
		nil, v,
		append(sshOptions, options...)...,
	)
}

func (c *RemoteCommand) Close() error {
	return c.Cmd.Close()
}

func NewRemoteCommand(s *Ssh, c Command) *RemoteCommand {
	return &RemoteCommand{
		Cmd: c,
		Ssh: s,
	}
}

var _ Command = &RemoteCommand{}

//

func CommandOptionEnv(env Environment) CommandOption {
	return func(cmd *Cmd) {
		cmd.Env = cmd.Env.With(env)
	}
}

//

func CommandExecute(command string, arguments []string, options ...CommandOption) ([]byte, error) {
	cmd := &Cmd{
		Cmd: exec.Command(command, arguments...),
		// NOTE: serves the requirement of multiple
		// CommandOptionEnv concatenation
		Env: NewEnvironment(),
	}
	for _, option := range options {
		option(cmd)
	}

	cmd.Cmd.Env = cmd.Env.Slice()
	cmd.Cmd.Stderr = nil // NOTE: required to get process Stderr
	for _, kv := range os.Environ() {
		keyValue := strings.SplitN(kv, "=", 2)
		_, exists := cmd.Env[keyValue[0]]
		if !exists {
			cmd.Cmd.Env = append(cmd.Cmd.Env, kv)
		}
	}

	output, err := cmd.Output()
	if err != nil {
		var stderr string
		exiterr, ok := err.(*exec.ExitError)
		if ok {
			stderr = string(exiterr.Stderr)
		}

		return nil, errors.Wrapf(
			err, "subcommand %q exited with: %s",
			command+" "+strings.Join(arguments, " "),
			stderr,
		)
	}
	return output, nil
}

func CommandExecuteUnmarshal(command string, arguments []string, unmarshaler Unmarshaler, result interface{}, options ...CommandOption) error {
	output, err := CommandExecute(command, arguments, options...)
	if err != nil {
		return err
	}

	if result != nil {
		if unmarshaler == nil {
			unmarshaler = NewUnmarshalerPassthrough()
		}

		err = unmarshaler.Unmarshal(output, &result)
		if err != nil {
			return err
		}
	}
	return nil
}

//

// TODO: by default we expect "Add" semantics
// but at some point in time we may need "Set" semantics
// so it probably should be injectable
func (e Environment) With(ee Environment) Environment {
	xs := e.Copy()
	for k, v := range ee {
		xs.Add(k, v...)
	}
	return xs
}

func (e Environment) Copy() Environment {
	xs := make(Environment, len(e))
	for k, v := range e {
		xs[k] = make([]string, len(v))
		copy(xs[k], v)
	}
	return xs
}

func (e Environment) Set(key string, value ...string) Environment {
	e[key] = value
	return e
}

func (e Environment) Add(key string, value ...string) Environment {
	e[key] = append(e[key], value...)
	return e
}

func (e Environment) Del(key string) Environment {
	delete(e, key)
	return e
}

func (e Environment) Slice() []string {
	xs := make([]string, len(e))
	n := 0
	for k := range e {
		xs[n] = k + "=" + strings.Join(e[k], " ")
		n++
	}
	return xs
}

func NewEnvironment(vars ...map[string][]string) Environment {
	e := Environment{}
	for _, vs := range vars {
		for k, v := range vs {
			e.Set(k, v...)
		}
	}
	return e
}
