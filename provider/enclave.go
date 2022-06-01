package provider

import (
	"github.com/awnumar/memguard"
)

type (
	Enclave      = memguard.Enclave
	LockedBuffer = memguard.LockedBuffer
)

func NewLockedBuffer(buf []byte) *LockedBuffer {
	return memguard.NewBufferFromBytes(buf)
}
