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
		Mode           NixMode
		Arguments      []string
		CommandOptions []CommandOption
		Environment    Environment
		Ssh            *Ssh
	}
	NixOption  func(*Nix)
	NixMode    uint8
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
	// NixMode exists to address some bugs in nix
	// - https://github.com/NixOS/nix/pull/6522
	//

	NixModeCompat  NixMode = 0
	NixModeDefault NixMode = 1
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
	options = append(options, n.CommandOptions...)
	if len(n.Environment) > 0 {
		options = append(options, CommandOptionEnv(n.Environment))
	}

	return "nix", arguments, options
}

func (n *Nix) Execute(result interface{}) error {
	command, arguments, options := n.Command()
	return CommandExecuteUnmarshal(command, arguments, nil, result, options...)
}

func (n *Nix) Build(options ...NixBuildCommandOption) *NixBuildCommand {
	command := &NixBuildCommand{Nix: n}
	for _, option := range options {
		option(command)
	}
	return command
}

func NixBuildCommandOptionFile(fd File) NixBuildCommandOption {
	return func(n *NixBuildCommand) {
		n.Arguments = append(n.Arguments, []string{"-f", fd.Name()}...)
	}
}

func NixBuildCommandOptionArg(name string, expr string) NixBuildCommandOption {
	return func(n *NixBuildCommand) {
		n.Arguments = append(n.Arguments, []string{"--arg", name, expr}...)
	}
}

func NixBuildCommandOptionArgStr(name string, value string) NixBuildCommandOption {
	return func(n *NixBuildCommand) {
		n.Arguments = append(n.Arguments, []string{"--argstr", name, value}...)
	}
}

func NixBuildCommandOptionNoLink() NixBuildCommandOption {
	return func(n *NixBuildCommand) {
		n.Arguments = append(n.Arguments, "--no-link")
	}
}

func NixBuildCommandOptionJSON() NixBuildCommandOption {
	return func(n *NixBuildCommand) {
		n.Arguments = append(n.Arguments, "--json")
		n.Unmarshaler = NewUnmarshalerJSON()
	}
}

func (n *NixBuildCommand) Command() (string, []string, []CommandOption) {
	command, arguments, options := n.Nix.Command()
	return command, append(append(arguments, "build"), n.Arguments...), options
}

func (n *NixBuildCommand) Execute(result interface{}) error {
	command, arguments, options := n.Command()
	return CommandExecuteUnmarshal(command, arguments, n.Unmarshaler, result, options...)
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

func NixCopyCommandOptionFrom(protocol NixCopyProtocol, from string) NixCopyCommandOption {
	return func(n *NixCopyCommand) {
		n.Arguments = append(n.Arguments, "--from", protocol.Path(from))
	}
}

func NixCopyCommandOptionPath(path string) NixCopyCommandOption {
	return func(n *NixCopyCommand) {
		n.Arguments = append(n.Arguments, path)
	}
}

func NixCopyCommandOptionUseSubstitutes() NixCopyCommandOption {
	return func(n *NixCopyCommand) {
		n.Arguments = append(n.Arguments, "--use-substitutes")
	}
}

func (n *NixCopyCommand) Command() (string, []string, []CommandOption) {
	command, arguments, options := n.Nix.Command()
	return command, append(append(arguments, "copy"), n.Arguments...), options
}

func (n *NixCopyCommand) Execute(result interface{}) error {
	command, arguments, options := n.Command()
	return CommandExecuteUnmarshal(command, arguments, nil, result, options...)
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

func (n *NixProfileCommand) Execute(result interface{}) error {
	command, arguments, options := n.Command()
	return CommandExecuteUnmarshal(command, arguments, nil, result, options...)
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
		switch n.Profile.Nix.Mode {
		case NixModeCompat:
			n.Arguments = append(n.Arguments, "--set", path)
		default:
			n.Arguments = append(n.Arguments, "--derivation", path)
		}
	}
}

func (n *NixProfileInstallCommand) Command() (string, []string, []CommandOption) {
	switch n.Profile.Nix.Mode {
	case NixModeCompat:
		_, _, options := n.Profile.Command()
		arguments := make([]string, len(n.Arguments))
		copy(arguments, n.Arguments)
		return "nix-env", arguments, options
	default:
		command, arguments, options := n.Profile.Command()
		return command, append(append(arguments, "install"), n.Arguments...), options
	}
}

func (n *NixProfileInstallCommand) Execute(result interface{}) error {
	command, arguments, options := n.Command()
	return CommandExecuteUnmarshal(command, arguments, nil, result, options...)
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

func NixOptionMode(mode NixMode) NixOption {
	return func(n *Nix) {
		n.Mode = mode
		switch mode {
		case NixModeCompat:
			NixOptionExperimentalFeatures(NixFeatureCommand)(n)
		}
	}
}

func NixOptionWithCommandOptions(options ...CommandOption) NixOption {
	return func(n *Nix) {
		n.CommandOptions = append(n.CommandOptions, options...)
	}
}

func NixOptionExperimentalFeatures(feature ...NixFeature) NixOption {
	return func(n *Nix) {
		switch n.Mode {
		case NixModeCompat:
			n.Arguments = append(
				n.Arguments, "--extra-experimental-features",
				strings.Join(feature, ","),
			)
		}
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
