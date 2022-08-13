package provider

import (
	"strings"
)

type (
	Ssh struct {
		Arguments  []string
		Finalizers []func()
	}
	SshOption         func(*Ssh)
	SshFinalizer      func(*Ssh)
	SshConfigKeyValue struct {
		Key   string
		Value string
	}
	SshConfigPairs = []SshConfigKeyValue
	SshConfigMap   struct {
		index map[string]int
		store SshConfigPairs
	}
)

func (m *SshConfigMap) Set(key, value string) {
	lkey := strings.ToLower(key)
	i, found := m.index[lkey]
	if found {
		m.store[i].Value = value
	} else {
		m.index[lkey] = len(m.store)
		m.store = append(m.store, SshConfigKeyValue{
			Key:   lkey,
			Value: value,
		})
	}
}

func (m *SshConfigMap) Get(key string) (string, bool) {
	i, found := m.index[strings.ToLower(key)]
	if !found {
		return "", false
	}
	return m.store[i].Value, true
}

func (m *SshConfigMap) Len() int {
	return len(m.store)
}

func (m *SshConfigMap) Pairs() SshConfigPairs {
	ps := make(SshConfigPairs, len(m.store))
	copy(ps, m.store)

	return ps
}

func NewSshConfigMap() *SshConfigMap {
	return &SshConfigMap{
		index: map[string]int{},
	}
}

const (
	SshConfigKeyHost         = "host"
	SshConfigKeyUser         = "user"
	SshConfigKeyPort         = "port"
	SshConfigKeyProxyCommand = "proxyCommand"
)

//

func SshSerializeConfig(ps SshConfigPairs) string {
	var config string
	for _, v := range ps {
		config += v.Key + " " + v.Value + "\n"
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

func SshOptionConfigMap(ps SshConfigPairs) SshOption {
	return func(s *Ssh) {
		fd, err := CreateTemp("ssh_config.*")
		if err != nil {
			panic(err)
		}
		_, err = fd.Write([]byte(SshSerializeConfig(ps)))
		if err != nil {
			panic(err)
		}

		SshFinalizerFile(fd)(s)
		SshOptionConfigFile(fd)(s)
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

func SshOptionNonInteractive() SshOption {
	return func(s *Ssh) {
		s.Arguments = append(s.Arguments, "-N")
	}
}

func SshOptionIORedirection(host, port string) SshOption {
	return func(s *Ssh) {
		s.Arguments = append(s.Arguments, "-W", host+":"+port)
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

func (s *Ssh) With(options ...SshOption) *Ssh {
	ss := &Ssh{
		Arguments:  make([]string, len(s.Arguments)),
		Finalizers: make([]func(), len(s.Finalizers)),
	}
	copy(ss.Arguments, s.Arguments)
	copy(ss.Finalizers, s.Finalizers)

	for _, option := range options {
		option(ss)
	}
	return ss
}

func (s *Ssh) Command() (string, []string, []CommandOption) {
	return "ssh", s.Arguments, nil
}

func (s *Ssh) Execute(result interface{}) error {
	defer s.Finalize()
	command, arguments, options := s.Command()
	return CommandExecuteUnmarshal(command, arguments, nil, result, options...)
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
