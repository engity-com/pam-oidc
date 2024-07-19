package sftp

import (
	"errors"
	"fmt"
	log "github.com/echocat/slf4g"
	"github.com/echocat/slf4g/level"
	"github.com/pkg/sftp"
	"io"
	"os"
)

type Server struct {
	Logger     log.Logger
	WorkingDir string
}

func (this *Server) Run(target io.ReadWriteCloser) error {
	s, err := sftp.NewServer(
		target,
		sftp.WithDebug(this.debugLogWriter()),
		sftp.WithServerWorkingDirectory(this.workingDir()),
	)
	if err != nil {
		return fmt.Errorf("cannot initialize sftp-server: %w", err)
	}
	defer func() { _ = s.Close() }()

	if err := s.Serve(); errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
		// Ignoring...
	} else if err != nil {
		return err
	}
	return nil
}

func (this *Server) debugLogWriter() io.Writer {
	return &log.LoggingWriter{
		Logger:         this.Logger,
		LevelExtractor: level.FixedLevelExtractor(level.Debug),
	}
}

func (this *Server) logger() log.Logger {
	if v := this.Logger; v != nil {
		return v
	}
	return log.GetRootLogger()
}

func (this *Server) workingDir() string {
	if v := this.WorkingDir; v != "" {
		return v
	}
	dir, err := os.Getwd()
	if err == nil {
		return dir
	}
	return "/"
}