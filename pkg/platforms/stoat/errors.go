package stoat

import (
	"strconv"

	"github.com/williamhorning/lightning/internal/rvapi"
	"github.com/williamhorning/lightning/pkg/lightning"
)

type stoatPermissionsError struct {
	permissions rvapi.Permission
	expected    rvapi.Permission
}

func (*stoatPermissionsError) Disable() *lightning.ChannelDisabled {
	return &lightning.ChannelDisabled{Read: false, Write: true}
}

func (e *stoatPermissionsError) Error() string {
	return "insufficient permissions (have " +
		strconv.FormatUint(uint64(e.permissions), 10) + ", want " +
		strconv.FormatUint(uint64(e.expected), 10) + "), please check them"
}

type stoatStatusError struct {
	msg  string
	resp []byte
	code int
	edit bool
}

func (e *stoatStatusError) Disable() *lightning.ChannelDisabled {
	return &lightning.ChannelDisabled{Read: false, Write: e.code == 401 || e.code == 403 || (e.code == 404 && !e.edit)}
}

func (e *stoatStatusError) Error() string {
	return "stoat status code " + strconv.FormatInt(int64(e.code), 10) + ": " + e.msg + ": " + string(e.resp)
}
