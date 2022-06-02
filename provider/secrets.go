package provider

import (
	"archive/tar"
	"bytes"
	"crypto/sha256"
	"io"
	"io/ioutil"
	"math"
	"os"
	"time"

	"encoding/hex"

	"github.com/pkg/errors"
	"golang.org/x/crypto/pbkdf2"
)

type (
	SecretDescription struct {
		Source      string
		Destination string
		Owner       string
		Group       string
		Permissions int
	}
	SecretData struct {
		*LockedBuffer
		*SecretDescription
	}
	SecretsDescriptions []*SecretDescription
	SecretsData         []*SecretData

	Secrets struct {
		Provider SecretsProvider
		Secrets  SecretsDescriptions
		data     SecretsData
	}

	SecretsProviderName string
	SecretsProvider     interface {
		Name() string
		Get(source string) ([]byte, error)
	}
	SecretsProviderFilesystem struct{}
	SecretsProviderCommand    struct {
		Command     string
		Arguments   []string
		Environment Environment
	}
	SecretsProviderGopass struct {
		*SecretsProviderCommand
	}

	SecretsCopy struct {
		*RemoteCommand
		Secrets *Secrets
	}
	SecretsCopyOption func(*SecretsCopy)
)

const SecretDataKeyLen = 32

const (
	SecretsProviderNameFilesystem SecretsProviderName = "filesystem"
	SecretsProviderNameCommand    SecretsProviderName = "command"
	SecretsProviderNameGopass     SecretsProviderName = "gopass"
)

var (
	SecretsProviders = []string{
		string(SecretsProviderNameFilesystem),
		string(SecretsProviderNameCommand),
		string(SecretsProviderNameGopass),
	}
)

//

func (s *SecretData) Hash(salt []byte, iter int) []byte {
	return pbkdf2.Key(
		s.Bytes(), salt, iter, SecretDataKeyLen,
		sha256.New,
	)
}

func (s *SecretData) HashString(salt []byte, iter int) string {
	return hex.EncodeToString(s.Hash(salt, iter))
}

//

func (s SecretsData) Hash(salt []byte, iter int) []byte {
	h := sha256.New()
	for _, secret := range s {
		_, _ = h.Write(secret.Hash(salt, iter))
	}
	return h.Sum(nil)
}

func (s SecretsData) HashString(salt []byte, iter int) string {
	return hex.EncodeToString(s.Hash(salt, iter))
}

func (s SecretsData) Destroy() {
	for _, secret := range s {
		secret.Destroy()
	}
}

//

func (p *SecretsProviderFilesystem) Name() string {
	return string(SecretsProviderNameFilesystem)
}

func (p *SecretsProviderFilesystem) Get(source string) ([]byte, error) {
	file, err := os.Open(source)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if stat.IsDir() {
		return nil, errors.Errorf(
			"got directory as secret at %q, this is not supported",
			source,
		)
	}

	buf, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, errors.Wrapf(
			err, "failed to read from file descriptor associated with %q",
			source,
		)
	}

	return buf, nil
}

func NewSecretsProviderFilesystem() *SecretsProviderFilesystem {
	return &SecretsProviderFilesystem{}
}

//

func (p *SecretsProviderCommand) Name() string {
	return string(SecretsProviderNameCommand)
}

func (p *SecretsProviderCommand) Get(source string) ([]byte, error) {
	arguments := make([]string, len(p.Arguments)+1)
	copy(arguments, p.Arguments)
	arguments[len(p.Arguments)] = source

	buf, err := CommandExecute(p.Command, arguments, CommandOptionEnv(p.Environment))
	if err != nil {
		return nil, errors.Wrapf(
			err, "failed to get content of %q using %q %v",
			source, p.Command, p.Arguments,
		)
	}

	return buf, nil
}

func NewSecretsProviderCommand(command string, arguments []string, environment map[string]string) *SecretsProviderCommand {
	env := NewEnvironment()
	for k, v := range environment {
		env.Set(k, v)
	}
	return &SecretsProviderCommand{
		Command:     command,
		Arguments:   arguments,
		Environment: env,
	}
}

//

func NewSecretsProviderGopass(store string) *SecretsProviderGopass {
	env := map[string]string{}
	if store != "" {
		env["PASSWORD_STORE_DIR"] = os.ExpandEnv(store)
	}

	return &SecretsProviderGopass{
		SecretsProviderCommand: NewSecretsProviderCommand("gopass", []string{"show", "-n"}, env),
	}
}

//

func (s *Secrets) fromOctal(n int) int {
	var (
		res = 0
		ctr = 0
		rem = 0
	)
	for n != 0 {
		rem = n % 10
		res += rem * int(math.Pow(8, float64(ctr)))
		n = n / 10
		ctr++
	}
	return res
}

func (s *Secrets) Tar() (io.Reader, error) {
	buf := bytes.NewBuffer(nil)
	w := tar.NewWriter(buf)
	defer w.Close()

	data, err := s.Data()
	if err != nil {
		return nil, err
	}

	for _, secret := range data {
		err := w.WriteHeader(&tar.Header{
			Name:    secret.Destination,
			Size:    int64(secret.Size()),
			Uname:   secret.Owner,
			Gname:   secret.Group,
			Mode:    int64(s.fromOctal(secret.Permissions)),
			ModTime: time.Now(),
		})
		if err != nil {
			return nil, errors.Wrapf(err, "failed to write tar header of %q", secret.Source)
		}

		_, err = w.Write(secret.Bytes())
		if err != nil {
			return nil, errors.Wrapf(err, "failed to write contents of %q into tar writer", secret.Source)
		}
	}

	return buf, nil
}

func (s *Secrets) Data() (SecretsData, error) {
	if s.data != nil {
		return s.data, nil
	}

	data := make(SecretsData, len(s.Secrets))
	for n, secret := range s.Secrets {
		buf, err := s.Provider.Get(secret.Source)
		if err != nil {
			// NOTE: destroy memory in case of error
			for _, secret := range data[:n] {
				secret.Destroy()
			}
			return nil, errors.Wrapf(
				err, "failed to get secret %q from provider %q",
				secret.Source, s.Provider.Name(),
			)
		}

		data[n] = &SecretData{
			LockedBuffer:      NewLockedBuffer(buf),
			SecretDescription: secret,
		}
	}

	s.data = data
	return data, nil
}

func (s *Secrets) Copy(ssh *Ssh) (*SecretsCopy, error) {
	stream, err := s.Tar()
	if err != nil {
		return nil, err
	}

	c := &SecretsCopy{
		RemoteCommand: NewRemoteCommand(ssh, NewTar(
			TarOptionExtract(),
			TarOptionChDir("/"),
			TarOptionCommandOptions(CommandOptionStdin(stream)),
		)),
		Secrets: s,
	}

	return c, nil
}

func (s *Secrets) Close() error {
	if s.data != nil {
		s.data.Destroy()
		s.data = nil
	}
	return nil
}

func NewSecrets(p SecretsProvider, s SecretsDescriptions) *Secrets {
	return &Secrets{
		Provider: p,
		Secrets:  s,
	}
}

var (
	_ Command = &SecretsCopy{}
)
