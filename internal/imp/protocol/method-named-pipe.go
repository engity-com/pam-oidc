package protocol

import (
	"context"
	gonet "net"
	"time"

	log "github.com/echocat/slf4g"
	"github.com/vmihailenco/msgpack/v5"
	"github.com/xtaci/smux"

	"github.com/engity-com/bifroest/pkg/codec"
	"github.com/engity-com/bifroest/pkg/common"
	"github.com/engity-com/bifroest/pkg/connection"
	"github.com/engity-com/bifroest/pkg/errors"
	"github.com/engity-com/bifroest/pkg/net"
	"github.com/engity-com/bifroest/pkg/sys"
)

type methodNamedPipeRequest struct {
	purpose net.Purpose
}

func (this methodNamedPipeRequest) EncodeMsgpack(enc *msgpack.Encoder) error {
	return this.EncodeMsgPack(enc)
}

func (this *methodNamedPipeRequest) DecodeMsgpack(dec *msgpack.Decoder) (err error) {
	return this.DecodeMsgPack(dec)
}

func (this methodNamedPipeRequest) EncodeMsgPack(enc codec.MsgPackEncoder) error {
	if b, err := this.purpose.MarshalText(); err != nil {
		return err
	} else if err := enc.EncodeBytes(b); err != nil {
		return err
	}
	return nil
}

func (this *methodNamedPipeRequest) DecodeMsgPack(dec codec.MsgPackDecoder) (err error) {
	if b, err := dec.DecodeBytes(); err != nil {
		return err
	} else if err := this.purpose.UnmarshalText(b); err != nil {
		return err
	}
	return nil
}

type methodNamedPipeResponse struct {
	path  string
	error error
}

func (this methodNamedPipeResponse) EncodeMsgpack(enc *msgpack.Encoder) error {
	return this.EncodeMsgPack(enc)
}

func (this *methodNamedPipeResponse) DecodeMsgpack(dec *msgpack.Decoder) (err error) {
	return this.DecodeMsgPack(dec)
}

func (this methodNamedPipeResponse) EncodeMsgPack(enc codec.MsgPackEncoder) error {
	if err := enc.EncodeString(this.path); err != nil {
		return err
	}
	if err := errors.EncodeMsgPack(this.error, enc); err != nil {
		return err
	}
	return nil
}

func (this *methodNamedPipeResponse) DecodeMsgPack(dec codec.MsgPackDecoder) (err error) {
	if this.path, err = dec.DecodeString(); err != nil {
		return err
	}
	if this.error, err = errors.DecodeMsgPack(dec); err != nil {
		return err
	}
	return nil
}

func (this *imp) handleMethodNamedPipe(ctx context.Context, header *Header, logger log.Logger, conn codec.MsgPackConn) error {
	failCore := func(err error) error {
		return errors.Network.Newf("handling %v failed: %w", header.Method, err)
	}
	failResponse := func(err error) error {
		rsp := methodNamedPipeResponse{error: err}
		if err := rsp.EncodeMsgPack(conn); err != nil {
			return failCore(err)
		}
		return nil
	}

	var req methodNamedPipeRequest
	if err := req.DecodeMsgPack(conn); err != nil {
		return failCore(err)
	}

	pipe, err := net.NewNamedPipe(req.purpose)
	if err != nil {
		return failResponse(err)
	}
	defer common.IgnoreCloseError(pipe)
	go func() {
		<-ctx.Done()
		_ = pipe.Close()
	}()

	conf := baseNamedPipeConfig()
	rsp := methodNamedPipeResponse{path: pipe.Path()}
	if err := rsp.EncodeMsgPack(conn); err != nil {
		return failCore(err)
	}

	nameOf := func(isL2r bool) string {
		if isL2r {
			return "source -> destination"
		}
		return "destination -> source"
	}

	in, out := gonet.Pipe()
	defer common.IgnoreCloseError(in)
	defer common.IgnoreCloseError(out)

	go func() {
		if err := sys.FullDuplexCopy(ctx, conn, in, &sys.FullDuplexCopyOpts{
			OnStart: func() {
				logger.Debug("port forwarding started")
			},
			OnEnd: func(s2d, d2s int64, duration time.Duration, err error, wasInL2r *bool) {
				ld := logger.
					With("s2d", s2d).
					With("d2s", d2s).
					With("duration", duration)
				if wasInL2r != nil {
					ld = ld.With("direction", nameOf(*wasInL2r))
				}

				if err != nil {
					ld.WithError(err).Error("cannot successful handle port forwarding request; canceling...")
				} else {
					ld.Info("port forwarding finished")
				}
			},
			OnStreamEnd: func(isL2r bool, err error) {
				defer common.IgnoreCloseError(pipe)
				name := "source -> destination"
				if !isL2r {
					name = "destination -> source"
				}
				logger.WithError(err).Tracef("coping of %s done", name)
			},
		}); err != nil {
			logger.WithError(err).
				Error("connection failed unexpectedly")
		}
	}()

	muxClient, err := smux.Client(out, conf)
	if err != nil {
		return failResponse(err)
	}

	for {
		pipeConn, err := pipe.AcceptNamedPipeConnection()
		if net.IsClosedError(err) {
			return nil
		} else if err != nil {
			return failResponse(err)
		}
		go func(pipeConn net.CloseWriterConn) struct{} {
			defer common.IgnoreCloseError(pipeConn)

			fail := func(err error) struct{} {
				logger.WithError(err).
					Error("connection failed unexpectedly")
				return struct{}{}
			}

			muxConn, err := muxClient.Open()
			if err != nil {
				return fail(err)
			}
			defer common.IgnoreCloseError(muxConn)

			if err := sys.FullDuplexCopy(ctx, pipeConn, muxConn, &sys.FullDuplexCopyOpts{
				OnStart: func() {
					logger.Debug("port forwarding started")
				},
				OnEnd: func(s2d, d2s int64, duration time.Duration, err error, wasInL2r *bool) {
					ld := logger.
						With("s2d", s2d).
						With("d2s", d2s).
						With("duration", duration)
					if wasInL2r != nil {
						ld = ld.With("direction", nameOf(*wasInL2r))
					}

					if err != nil {
						ld.WithError(err).Error("cannot successful handle port forwarding request; canceling...")
					} else {
						ld.Info("port forwarding finished")
					}
				},
				OnStreamEnd: func(isL2r bool, err error) {
					name := "source -> destination"
					if !isL2r {
						name = "destination -> source"
					}
					logger.WithError(err).Tracef("coping of %s done", name)
				},
			}); err != nil {
				return fail(err)
			}

			return struct{}{}
		}(pipeConn)
	}
}

func (this *Master) methodNamedPipe(ctx context.Context, ref Ref, connectionId connection.Id, purpose net.Purpose) (net.NamedPipe, error) {
	fail := func(err error) (net.NamedPipe, error) {
		return nil, errors.Network.Newf("handling %v failed: %w", MethodNamedPipe, err)
	}

	success := false
	conn, err := this.DialContextWithMsgPack(ctx, ref)
	if err != nil {
		return fail(err)
	}
	defer common.IgnoreCloseErrorIfFalse(&success, conn)

	if err := (Header{MethodNamedPipe, connectionId}).EncodeMsgPack(conn); err != nil {
		return fail(err)
	}

	if err := (methodNamedPipeRequest{purpose}).EncodeMsgPack(conn); err != nil {
		return fail(err)
	}

	var rsp methodNamedPipeResponse
	if err := rsp.DecodeMsgPack(conn); err != nil {
		return fail(err)
	}

	conf := baseNamedPipeConfig()
	muxServer, err := smux.Server(conn, conf)
	if err != nil {
		return fail(err)
	}
	defer common.IgnoreCloseErrorIfFalse(&success, muxServer)

	ln, err := net.AsListener(muxServer.OpenStream, muxServer.Close, func() string {
		return rsp.path
	})
	if err != nil {
		return fail(err)
	}
	defer common.IgnoreCloseErrorIfFalse(&success, ln)

	pipe, err := net.AsNamedPipe(ln, rsp.path)
	if err != nil {
		return fail(err)
	}
	defer common.IgnoreCloseErrorIfFalse(&success, pipe)

	success = true
	return pipe, err
}

func baseNamedPipeConfig() *smux.Config {
	result := smux.DefaultConfig()
	result.Version = 2
	return result
}
