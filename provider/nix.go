package provider

import (
	"crypto/sha1"
	"encoding/hex"
	"net/url"
	"strconv"
	"strings"
)

type (
	Nix struct {
		Arguments   []string
		Environment Environment
		Ssh         *Ssh
	}
	NixOption  func(*Nix)
	NixFeature = string

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

	NixProfileCommand struct {
		Nix       *Nix
		Arguments []string
	}
	NixProfileCommandOption func(*NixProfileCommand)

	NixProfileInstallCommand struct {
		Profile   *NixProfileCommand
		Arguments []string
	}
	NixProfileInstallCommandOption func(*NixProfileInstallCommand)

	//

	NixActivationAction = string

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
	NixFeatureCommand NixFeature = "nix-command"
)

const (
	NixCopyProtocolNone NixCopyProtocol = ""
	NixCopyProtocolSSH  NixCopyProtocol = "ssh"
	NixCopyProtocolS3   NixCopyProtocol = "s3"
	NixCopyProtocolFile NixCopyProtocol = "file"
)

const (
	NixActivationActionNone        NixActivationAction = ""
	NixActivationActionSwitch      NixActivationAction = "switch"
	NixActivationActionBoot        NixActivationAction = "boot"
	NixActivationActionTest        NixActivationAction = "test"
	NixActivationActionDryActivate NixActivationAction = "dry-activate"
)

func (n NixCopyProtocol) Path(path string) string {
	u := &url.URL{}
	if len(n) > 0 {
		u.Scheme = string(n)
	}
	u.Path = path
	return u.String()
}

//

func (n *Nix) With(options ...NixOption) *Nix {
	nn := &Nix{
		Arguments:   make([]string, len(n.Arguments)),
		Environment: n.Environment.Copy(),
		Ssh:         n.Ssh.With(),
	}
	for n, v := range n.Arguments {
		nn.Arguments[n] = v
	}

	for _, option := range options {
		option(nn)
	}
	return nn
}

func (n *Nix) Command() (string, []string, []CommandOption) {
	arguments := make([]string, len(n.Arguments))
	copy(arguments, n.Arguments)

	options := []CommandOption{}
	if len(n.Environment) > 0 {
		options = append(options, CommandOptionEnv(n.Environment))
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

func (n *NixBuildCommand) Command() (string, []string, []CommandOption) {
	command, arguments, options := n.Nix.Command()
	return command, append(append(arguments, "build"), n.Arguments...), options
}

func (n *NixBuildCommand) Execute(v interface{}) error {
	command, arguments, options := n.Command()
	return CommandExecuteUnmarshal(command, arguments, n.Unmarshaler, v, options...)
}

func (n *NixBuildCommand) Close() error { return nil }

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

func (n *NixCopyCommand) Command() (string, []string, []CommandOption) {
	command, arguments, options := n.Nix.Command()
	return command, append(append(arguments, "copy"), n.Arguments...), options
}

func (n *NixCopyCommand) Execute(v interface{}) error {
	command, arguments, options := n.Command()
	return CommandExecuteUnmarshal(command, arguments, nil, v, options...)
}

func (n *NixCopyCommand) Close() error { return nil }

//

func (n *Nix) Profile(options ...NixProfileCommandOption) *NixProfileCommand {
	command := &NixProfileCommand{Nix: n}
	for _, option := range options {
		option(command)
	}
	return command
}

func (n *NixProfileCommand) Command() (string, []string, []CommandOption) {
	command, arguments, options := n.Nix.Command()
	return command, append(append(arguments, "profile"), n.Arguments...), options
}

func (n *NixProfileCommand) Execute(v interface{}) error {
	command, arguments, options := n.Command()
	return CommandExecuteUnmarshal(command, arguments, nil, v, options...)
}

func (n *NixProfileCommand) Close() error { return nil }

//

func (n *NixProfileCommand) Install(options ...NixProfileInstallCommandOption) *NixProfileInstallCommand {
	command := &NixProfileInstallCommand{Profile: n}
	for _, option := range options {
		option(command)
	}
	return command
}

func NixProfileInstallCommandOptionProfile(location string) NixProfileInstallCommandOption {
	return func(n *NixProfileInstallCommand) {
		n.Arguments = append(n.Arguments, "--profile", location)
	}
}

func NixProfileInstallCommandOptionDerivation(path string) NixProfileInstallCommandOption {
	return func(n *NixProfileInstallCommand) {
		n.Arguments = append(n.Arguments, "--derivation", path)
	}
}

func (n *NixProfileInstallCommand) Command() (string, []string, []CommandOption) {
	command, arguments, options := n.Profile.Command()
	return command, append(append(arguments, "install"), n.Arguments...), options
}

func (n *NixProfileInstallCommand) Execute(v interface{}) error {
	command, arguments, options := n.Command()
	return CommandExecuteUnmarshal(command, arguments, nil, v, options...)
}

func (n *NixProfileInstallCommand) Close() error { return nil }

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
		n.Environment.Add(NixEnvironmentSshOpts, opts...)
	}
}

func NixOptionExperimentalFeatures(feature ...NixFeature) NixOption {
	return func(n *Nix) {
		n.Arguments = append(
			n.Arguments, "--extra-experimental-features",
			strings.Join(feature, ","),
		)
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
		Environment: Environment{},
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
