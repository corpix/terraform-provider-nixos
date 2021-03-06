package provider

import (
	"bufio"
	"bytes"
	"context"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/pkg/errors"
)

type (
	Cmd struct {
		*exec.Cmd
		PreRunHooks []func(*Cmd)
		Stdout      *bytes.Buffer
		Stderr      *bytes.Buffer
		Env         Environment
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
		Ssh *Ssh
		Cmd Command
	}

	Environment map[string][]string
)

func Readln(r *bufio.Reader) ([]byte, error) {
	var (
		isPrefix bool = true
		err      error
		line, ln []byte
	)
	for isPrefix && err == nil {
		line, isPrefix, err = r.ReadLine()
		ln = append(ln, line...)
	}
	return ln, err
}

func (c *Cmd) Run() ([]byte, []byte, error) {
	for _, hook := range c.PreRunHooks {
		hook(c)
	}

	err := c.Cmd.Run()

	return c.Stdout.Bytes(), c.Stderr.Bytes(), err
}

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
		Ssh: s,
		Cmd: c,
	}
}

var _ Command = &RemoteCommand{}

//

func CommandOptionStdin(stdin io.Reader) CommandOption {
	return func(cmd *Cmd) {
		cmd.Stdin = stdin
	}
}

func CommandOptionEnv(env Environment) CommandOption {
	return func(cmd *Cmd) {
		cmd.Env = cmd.Env.With(env)
	}
}

func CommandOptionTflogTee(ctx context.Context) CommandOption {
	logWriter := NewLogWriter(ctx)
	return func(cmd *Cmd) {
		cmd.Cmd.Stdout = io.MultiWriter(cmd.Stdout, logWriter)
		cmd.Cmd.Stderr = io.MultiWriter(cmd.Stderr, logWriter)
		cmd.PreRunHooks = append(
			cmd.PreRunHooks,
			func(cmd *Cmd) { tflog.Info(ctx, "running command: "+cmd.String()) },
		)
	}
}

//

func CommandExecute(command string, arguments []string, options ...CommandOption) ([]byte, error) {
	execCmd := exec.Command(command, arguments...)
	stdout := bytes.NewBuffer(nil)
	stderr := bytes.NewBuffer(nil)
	execCmd.Stdout = stdout
	execCmd.Stderr = stderr

	//

	cmd := &Cmd{
		Cmd: execCmd,
		// NOTE: serves the requirement of multiple
		// CommandOptionEnv concatenation
		Stdout: stdout,
		Stderr: stderr,
		Env:    NewEnvironment(),
	}
	for _, option := range options {
		option(cmd)
	}

	cmd.Cmd.Env = cmd.Env.Slice()
	for _, kv := range os.Environ() {
		keyValue := strings.SplitN(kv, "=", 2)
		_, exists := cmd.Env[keyValue[0]]
		if !exists {
			cmd.Cmd.Env = append(cmd.Cmd.Env, kv)
		}
	}

	stdoutBytes, stderrBytes, err := cmd.Run()
	if err != nil {
		return nil, errors.Wrapf(
			err, "subcommand %q exited with: %s",
			cmd.String(),
			string(stderrBytes),
		)
	}
	return stdoutBytes, nil
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
