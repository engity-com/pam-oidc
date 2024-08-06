//go:build unix

package user

import (
	"errors"
	"fmt"
	"strconv"
	"time"
)

const (
	etcPasswdColons = 7
)

var (
	DefaultEtcPasswd = "/etc/passwd"

	errEtcPasswdEmptyUid       = errors.New("empty UID")
	errEtcPasswdIllegalUid     = errors.New("illegal UID")
	errEtcPasswdEmptyGid       = errors.New("empty GID")
	errEtcPasswdIllegalGid     = errors.New("illegal GID")
	errEtcPasswdEmptyHomeDir   = errors.New("empty home directory")
	errEtcPasswdTooLongHomeDir = errors.New("home directory is longer than 255 characters")
	errEtcPasswdIllegalHomeDir = errors.New("illegal home directory")
	errEtcPasswdEmptyShell     = errors.New("empty shell")
	errEtcPasswdTooLongShell   = errors.New("shell is longer than 255 characters")
	errEtcPasswdIllegalShell   = errors.New("illegal shell")
)

type etcPasswdEntry struct {
	name     []byte
	password []byte
	uid      uint32
	gid      uint32
	geocs    []byte
	homeDir  []byte
	shell    []byte
}

func (this *etcPasswdEntry) validate(allowBadName bool) error {
	if err := validateUserName(this.name, allowBadName); err != nil {
		return err
	}
	if err := validateGeocs(this.geocs); err != nil {
		return err
	}
	if err := validateColonFilePathColumn(this.homeDir, errEtcPasswdEmptyHomeDir, errEtcPasswdTooLongHomeDir, errEtcPasswdIllegalHomeDir); err != nil {
		return err
	}
	if err := validateColonFilePathColumn(this.shell, errEtcPasswdEmptyShell, errEtcPasswdTooLongShell, errEtcPasswdIllegalShell); err != nil {
		return err
	}
	return nil
}

func (this *etcPasswdEntry) decode(line [][]byte, allowBadName bool) error {
	var err error
	this.name = line[0]
	this.password = line[1]
	if this.uid, _, err = parseUint32Column(line, 2, errEtcPasswdEmptyUid, errEtcPasswdIllegalUid); err != nil {
		return err
	}
	if this.gid, _, err = parseUint32Column(line, 3, errEtcPasswdEmptyGid, errEtcPasswdIllegalGid); err != nil {
		return err
	}
	this.geocs = line[4]
	this.homeDir = line[5]
	this.shell = line[6]

	if err := this.validate(allowBadName); err != nil {
		return err
	}

	return nil
}

func (this *etcPasswdEntry) encode(allowBadName bool) ([][]byte, error) {
	if err := this.validate(allowBadName); err != nil {
		return nil, err
	}

	line := make([][]byte, 7)
	line[0] = this.name
	line[1] = this.password
	line[2] = []byte(strconv.FormatUint(uint64(this.uid), 10))
	line[3] = []byte(strconv.FormatUint(uint64(this.gid), 10))
	line[4] = this.geocs
	line[5] = this.homeDir
	line[6] = this.shell

	return line, nil

}

type etcPasswdRef struct {
	*etcPasswdEntry
	*etcShadowEntry
}

func (this *Requirement) toEtcPasswdRef(gui GroupId, idGenerator func() (Id, error)) (*etcPasswdRef, error) {
	result := etcPasswdRef{
		&etcPasswdEntry{
			[]byte{},
			[]byte("x"),
			uint32(0),
			uint32(gui),
			[]byte(this.DisplayName),
			[]byte(this.HomeDir),
			[]byte(this.Shell),
		},
		&etcShadowEntry{
			[]byte{},
			[]byte("*"),
			uint64(time.Now().Unix()),
			0,
			99999,
			7, true,
			0, false,
			0, false,
		},
	}

	if v := this.Uid; v != nil {
		result.uid = uint32(*v)
	} else if v, err := idGenerator(); err != nil {
		return nil, err
	} else {
		result.uid = uint32(v)
	}

	if v := this.Name; v != "" {
		result.etcPasswdEntry.name = []byte(v)
	} else {
		result.etcPasswdEntry.name = []byte(fmt.Sprintf("u%d", result.etcPasswdEntry.uid))
	}
	result.etcShadowEntry.name = result.etcPasswdEntry.name

	return &result, nil
}

func (this *Requirement) updateEtcPasswdRef(ref *etcPasswdRef, gui GroupId) error {
	if v := this.Uid; v != nil {
		ref.etcPasswdEntry.uid = uint32(*v)
	}
	ref.etcPasswdEntry.gid = uint32(gui)

	if v := this.Name; v != "" {
		ref.etcPasswdEntry.name = []byte(v)
		ref.etcShadowEntry.name = ref.etcPasswdEntry.name
	}

	ref.etcPasswdEntry.geocs = []byte(this.DisplayName)
	ref.etcPasswdEntry.shell = []byte(this.Shell)
	ref.etcPasswdEntry.homeDir = []byte(this.HomeDir) //TODO! We should consider to move it?

	return nil
}

type nameToEtcPasswdRef map[string]*etcPasswdRef
type idToEtcPasswdRef map[Id]*etcPasswdRef