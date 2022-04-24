package provider

import (
	_ "embed"
	"os"
)

//go:embed nix_conf_wrapper.nix
var NixWrapper []byte

func NewNixWrapperFile(path string) (File, error) {
	var (
		fd  File
		err error
	)
	if path == "" {
		fd, err = CreateTemp("nix_wrapper.nix.*")
		if err != nil {
			return nil, err
		}
		_, err = fd.Write(NixWrapper)
		if err != nil {
			fd.Close()
			return nil, err
		}
	} else {
		fd, err = os.Open(path)
		if err != nil {
			return nil, err
		}
	}
	return fd, nil
}
