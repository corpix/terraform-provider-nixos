package provider

import (
	"crypto/sha1"
	"encoding/hex"
	"net/url"
	"strconv"
)

type (
	Nix struct {
		Arguments []string
		Env       Environment
		Ssh       *Ssh
	}
	NixOption func(*Nix)

	//

	NixBuildCommand struct {
		Nix       *Nix
		Arguments []string
		Unmarshaler
	}
	NixBuildCommandOption func(*NixBuildCommand)

	//

	NixCopyProtocol string
	NixCopyCommand  struct {
		Nix       *Nix
		Arguments []string
	}
	NixCopyCommandOption func(*NixCopyCommand)

	//

	Derivation struct {
		Path    string            `json:"drvPath" mapstructure:"path"`
		Outputs map[string]string `json:"outputs" mapstructure:"outputs"`
	}
	Derivations []Derivation
)

const (
	NixEnvironmentSshOpts = "NIX_SSHOPTS"
)

const (
	NixCopyProtocolNone NixCopyProtocol = ""
	NixCopyProtocolSSH  NixCopyProtocol = "ssh"
	NixCopyProtocolS3   NixCopyProtocol = "s3"
	NixCopyProtocolFile NixCopyProtocol = "file"
)

func (p NixCopyProtocol) Path(path string) string {
	u := &url.URL{}
	if len(p) > 0 {
		u.Scheme = string(p)
	}
	u.Path = path
	return u.String()
}

//

func (n *Nix) Command() (string, []string, []CommandOption) {
	arguments := make([]string, len(n.Arguments))
	copy(arguments, n.Arguments)

	options := []CommandOption{}
	if len(n.Env) > 0 {
		options = append(options, CommandOptionEnv(n.Env))
	}

	return "nix", arguments, options
}

func (n *Nix) Execute(v interface{}) error {
	command, arguments, options := n.Command()
	return CommandExecuteUnmarshal(command, arguments, nil, v, options...)
}

func (n *Nix) Build(options ...NixBuildCommandOption) *NixBuildCommand {
	command := &NixBuildCommand{Nix: n}
	for _, option := range options {
		option(command)
	}
	return command
}

func NixBuildCommandOptionFile(fd File) NixBuildCommandOption {
	return func(b *NixBuildCommand) {
		b.Arguments = append(b.Arguments, []string{"-f", fd.Name()}...)
	}
}

func NixBuildCommandOptionArg(name string, expr string) NixBuildCommandOption {
	return func(b *NixBuildCommand) {
		b.Arguments = append(b.Arguments, []string{"--arg", name, expr}...)
	}
}

func NixBuildCommandOptionArgStr(name string, value string) NixBuildCommandOption {
	return func(b *NixBuildCommand) {
		b.Arguments = append(b.Arguments, []string{"--argstr", name, value}...)
	}
}

func NixBuildCommandOptionNoLink() NixBuildCommandOption {
	return func(b *NixBuildCommand) {
		b.Arguments = append(b.Arguments, "--no-link")
	}
}

func NixBuildCommandOptionJSON() NixBuildCommandOption {
	return func(b *NixBuildCommand) {
		b.Arguments = append(b.Arguments, "--json")
		b.Unmarshaler = NewUnmarshalerJSON()
	}
}

func (b *NixBuildCommand) Command() (string, []string, []CommandOption) {
	command, arguments, options := b.Nix.Command()
	return command, append(append(arguments, "build"), b.Arguments...), options
}

func (b *NixBuildCommand) Execute(v interface{}) error {
	command, arguments, options := b.Command()
	return CommandExecuteUnmarshal(command, arguments, b.Unmarshaler, v, options...)
}

func (b *NixBuildCommand) Close() error { return nil }

//

func (n *Nix) Copy(options ...NixCopyCommandOption) *NixCopyCommand {
	command := &NixCopyCommand{Nix: n}
	for _, option := range options {
		option(command)
	}
	return command
}

func NixCopyCommandOptionTo(protocol NixCopyProtocol, to string) NixCopyCommandOption {
	return func(b *NixCopyCommand) {
		b.Arguments = append(b.Arguments, "--to", protocol.Path(to))
	}
}

func NixCopyCommandOptionPath(path string) NixCopyCommandOption {
	return func(b *NixCopyCommand) {
		b.Arguments = append(b.Arguments, path)
	}
}

func NixCopyCommandOptionFrom(protocol NixCopyProtocol, from string) NixCopyCommandOption {
	return func(b *NixCopyCommand) {
		b.Arguments = append(b.Arguments, "--from", protocol.Path(from))
	}
}

func NixCopyCommandOptionUseSubstitutes() NixCopyCommandOption {
	return func(b *NixCopyCommand) {
		b.Arguments = append(b.Arguments, "--use-substitutes")
	}
}

func (c *NixCopyCommand) Command() (string, []string, []CommandOption) {
	command, arguments, options := c.Nix.Command()
	return command, append(append(arguments, "copy"), c.Arguments...), options
}

func (c *NixCopyCommand) Execute(v interface{}) error {
	command, arguments, options := c.Command()
	return CommandExecuteUnmarshal(command, arguments, nil, v, options...)
}

func (b *NixCopyCommand) Close() error { return nil }

//

func NixOptionSsh(s *Ssh) NixOption {
	_, opts, _ := s.Command()
	setOpts := NixOptionSshOpts(opts...)
	return func(n *Nix) {
		n.Ssh = s
		setOpts(n)
	}
}

func NixOptionSshOpts(opts ...string) NixOption {
	return func(n *Nix) {
		n.Env.Add(NixEnvironmentSshOpts, opts...)
	}
}

func NixOptionShowTrace() NixOption {
	return func(n *Nix) {
		n.Arguments = append(n.Arguments, "--show-trace")
	}
}

func NixOptionCores(num int) NixOption {
	return func(n *Nix) {
		n.Arguments = append(n.Arguments, "--cores", strconv.Itoa(num))
	}
}

func NixOptionUseSubstitutes() NixOption {
	return func(n *Nix) {
		n.Arguments = append(n.Arguments, "--builders-use-substitutes")
	}
}

func (n *Nix) Close() error {
	var err error
	if n.Ssh != nil {
		err = n.Ssh.Close()
	}
	return err
}

func NewNix(options ...NixOption) *Nix {
	command := &Nix{
		Env: Environment{},
	}
	for _, option := range options {
		option(command)
	}
	return command
}

//

var (
	_ Command = NewNix()
	_ Command = NewNix().Build()
	_ Command = NewNix().Copy()
)

//

func (ds Derivations) Hash() string {
	var (
		hash = sha1.New()
		err  error
	)
	for _, d := range ds {
		_, err = hash.Write([]byte(d.Hash()))
		if err != nil {
			panic(err)
		}
	}
	return hex.EncodeToString(hash.Sum(nil))
}

func (d Derivation) Hash() string {
	var (
		hash = sha1.New()
		err  error
	)
	_, err = hash.Write([]byte(d.Path))
	if err != nil {
		panic(err)
	}
	return hex.EncodeToString(hash.Sum(nil))
}
