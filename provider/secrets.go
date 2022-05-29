package provider

import (
	"archive/tar"
	"bytes"
	"io"
	"math"
	"os"

	"github.com/pkg/errors"
)

type (
	Secret struct {
		Source      string
		Destination string
		Owner       string
		Group       string
		Permissions int
	}

	Secrets struct {
		secrets []*Secret
	}
	SecretsCopy struct {
		*RemoteCommand
		Secrets *Secrets
	}
	SecretsCopyOption func(*SecretsCopy)
)

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

func (s *Secrets) Stream() (io.Reader, error) {
	buf := bytes.NewBuffer(nil)
	w := tar.NewWriter(buf)
	defer w.Close()

	for _, secret := range s.secrets {
		file, err := os.Open(secret.Source)
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
				"got directory as secret at %q, this is not yet supported",
				secret.Source,
			)
		}

		err = w.WriteHeader(&tar.Header{
			Name:    secret.Destination,
			Size:    stat.Size(),
			Uname:   secret.Owner,
			Gname:   secret.Group,
			Mode:    int64(s.fromOctal(secret.Permissions)),
			ModTime: stat.ModTime(),
		})
		if err != nil {
			return nil, errors.Wrapf(err, "failed to write tar header of %q", secret.Source)
		}

		_, err = io.Copy(w, file)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to copy contents of %q into tar writer", secret.Source)
		}
	}

	return buf, nil
}

func (s *Secrets) Copy(ssh *Ssh) (*SecretsCopy, error) {
	stream, err := s.Stream()
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

func NewSecrets(secrets []*Secret) *Secrets {
	return &Secrets{secrets: secrets}
}

var (
	_ Command = &SecretsCopy{}
)
