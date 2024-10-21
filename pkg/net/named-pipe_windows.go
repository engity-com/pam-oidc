//go:build windows

package net

import (
	"fmt"

	"github.com/Microsoft/go-winio"
)

const (
	namedPipePrefix = `\\.\pipe\bifroest-`
)

func newNamedPipe(purpose Purpose, id string) (NamedPipe, error) {
	path := fmt.Sprintf("%s%v-%s", namedPipePrefix, purpose, id)

	c := winio.PipeConfig{
		SecurityDescriptor: "",
		MessageMode:        true,
		InputBufferSize:    65536,
		OutputBufferSize:   65536,
	}

	ln, err := winio.ListenPipe(path, &c)
	if err != nil {
		return nil, err
	}
	return &namedPipe{ln, path}, nil
}
