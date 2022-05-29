package provider

type (
	Tar struct {
		CommandOptions []CommandOption
		Arguments      []string
	}
	TarOption func(t *Tar)
)

func TarOptionCommandOptions(option ...CommandOption) TarOption {
	return func(t *Tar) {
		t.CommandOptions = append(t.CommandOptions, option...)
	}
}

func TarOptionCreate() TarOption {
	return func(t *Tar) {
		t.Arguments = append(t.Arguments, "-c")
	}
}

func TarOptionExtract() TarOption {
	return func(t *Tar) {
		t.Arguments = append(t.Arguments, "-x")
	}
}

func TarOptionGzip() TarOption {
	return func(t *Tar) {
		t.Arguments = append(t.Arguments, "-z")
	}
}

func TarOptionFile(path string) TarOption {
	return func(t *Tar) {
		t.Arguments = append(t.Arguments, "-f", path)
	}
}

func TarOptionChDir(path string) TarOption {
	return func(t *Tar) {
		t.Arguments = append(t.Arguments, "-C", path)
	}
}

func TarOptionPaths(paths ...string) TarOption {
	return func(t *Tar) {
		t.Arguments = append(t.Arguments, paths...)
	}
}

//

func (t *Tar) Command() (string, []string, []CommandOption) {
	arguments := make([]string, len(t.Arguments))
	copy(arguments, t.Arguments)
	return "tar", arguments, t.CommandOptions
}

func (t *Tar) Execute(result interface{}) error {
	command, arguments, options := t.Command()
	return CommandExecuteUnmarshal(command, arguments, nil, result, options...)
}

func (t *Tar) Close() error { return nil }

func NewTar(options ...TarOption) *Tar {
	t := &Tar{}
	for _, option := range options {
		option(t)
	}
	return t
}

var (
	_ Command = NewTar()
)
