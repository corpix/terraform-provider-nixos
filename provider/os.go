package provider

import (
	"os"
)

type File interface {
	Name() string
	Seek(offset int64, whence int) (ret int64, err error)
	Read(b []byte) (n int, err error)
	Write(b []byte) (n int, err error)
	Close() error
}

type TempFile struct {
	*os.File
}

func (fd TempFile) Close() error {
	_ = fd.File.Close()
	return os.Remove(fd.Name())
}

func CreateTemp(name string) (*TempFile, error) {
	fd, err := os.CreateTemp("", name)
	if err != nil {
		return nil, err
	}
	return &TempFile{File: fd}, nil
}

var (
	_ File = TempFile{}
)
