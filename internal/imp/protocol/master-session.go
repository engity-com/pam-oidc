package protocol

import (
	"context"
	gonet "net"

	"github.com/engity-com/bifroest/pkg/connection"
	"github.com/engity-com/bifroest/pkg/errors"
	"github.com/engity-com/bifroest/pkg/net"
	"github.com/engity-com/bifroest/pkg/sys"
)

type MasterSession struct {
	parent *Master
	ref    Ref
}

func (this *MasterSession) Close() error {
	return nil
}

func (this *MasterSession) InitiateTcpForward(ctx context.Context, connectionId connection.Id, target net.HostPort) (gonet.Conn, error) {
	fail := func(err error) (gonet.Conn, error) {
		return nil, errors.Network.Newf("cannot initiate direct tcp connection for %v to %v: %w", connectionId, target, err)
	}

	result, err := this.parent.methodTcpForward(ctx, this.ref, connectionId, target)
	if err != nil {
		return fail(err)
	}

	return result, nil
}

func (this *MasterSession) InitiateNamedPipe(ctx context.Context, connectionId connection.Id, purpose net.Purpose) (net.NamedPipe, error) {
	fail := func(err error) (net.NamedPipe, error) {
		return nil, errors.Network.Newf("cannot initiate direct tcp connection for %v of %v: %w", connectionId, purpose, err)
	}

	result, err := this.parent.methodNamedPipe(ctx, this.ref, connectionId, purpose)
	if err != nil {
		return fail(err)
	}

	return result, nil
}

func (this *MasterSession) Echo(ctx context.Context, connectionId connection.Id, in string) (string, error) {
	return this.parent.methodEcho(ctx, this.ref, connectionId, in)
}

func (this *MasterSession) Kill(ctx context.Context, connectionId connection.Id, pid int, signal sys.Signal) error {
	return this.parent.methodKill(ctx, this.ref, connectionId, pid, signal)
}

func (this *MasterSession) Exit(ctx context.Context, connectionId connection.Id, exitCode int) error {
	return this.parent.methodExit(ctx, this.ref, connectionId, exitCode)
}
