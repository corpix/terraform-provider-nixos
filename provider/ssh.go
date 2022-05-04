package provider

type (
	Ssh struct {
		Arguments  []string
		Finalizers []func()
	}
	SshOption    func(*Ssh)
	SshFinalizer func(*Ssh)
)

func SshSerializeConfig(m map[string]string) string {
	var config string
	for k, v := range m {
		config += k + " " + v + "\n"
	}
	return config
}

//

func SshOptionConfig(path string) SshOption {
	return func(s *Ssh) {
		s.Arguments = append(s.Arguments, "-F", path)
	}
}

func SshOptionConfigFile(fd File) SshOption {
	return SshOptionConfig(fd.Name())
}

func SshOptionConfigMap(m map[string]string) SshOption {
	return func(s *Ssh) {
		fd, err := CreateTemp("ssh_config.*")
		if err != nil {
			panic(err)
		}
		_, err = fd.Write([]byte(SshSerializeConfig(m)))
		if err != nil {
			panic(err)
		}

		SshFinalizerFile(fd)(s)
		SshOptionConfigFile(fd)(s)
	}
}

func SshOptionUser(user string) SshOption {
	return func(s *Ssh) {
		s.Arguments = append(s.Arguments, user)
	}
}

func SshOptionHost(host string) SshOption {
	return func(s *Ssh) {
		s.Arguments = append(s.Arguments, host)
	}
}

func SshOptionCommand(cmd string) SshOption {
	return func(s *Ssh) {
		s.Arguments = append(s.Arguments, cmd)
	}
}

//

func SshFinalizerFile(fd File) SshFinalizer {
	return func(s *Ssh) {
		s.Finalizers = append(
			s.Finalizers,
			func() { fd.Close() },
		)
	}
}

//

func (s *Ssh) Command() (string, []string, []CommandOption) {
	return "ssh", s.Arguments, nil
}

func (s *Ssh) Execute(v interface{}) error {
	defer s.Finalize()
	command, arguments, options := s.Command()
	return CommandExecuteUnmarshal(command, arguments, nil, v, options...)
}

func (s *Ssh) Finalize() {
	for _, finalize := range s.Finalizers {
		finalize()
	}
	s.Finalizers = nil
}

func (s *Ssh) Close() error {
	s.Finalize()
	return nil
}

func NewSsh(options ...SshOption) *Ssh {
	s := &Ssh{}
	for _, option := range options {
		option(s)
	}
	return s
}

var (
	_ Command = NewSsh()
)
