package revolt

import (
	"strconv"

	"github.com/williamhorning/lightning/internal/rvapi"
	"github.com/williamhorning/lightning/pkg/lightning"
)

type revoltPermissionsError struct {
	permissions rvapi.Permission
	expected    rvapi.Permission
}

func (*revoltPermissionsError) Disable() *lightning.ChannelDisabled {
	return &lightning.ChannelDisabled{Read: false, Write: true}
}

func (e *revoltPermissionsError) Error() string {
	return "insufficient permissions in Revolt (have " +
		strconv.FormatUint(uint64(e.permissions), 10) + ", want " +
		strconv.FormatUint(uint64(e.expected), 10) + "), please check them"
}

type revoltStatusError struct {
	msg  string
	code int
	edit bool
}

func (e *revoltStatusError) Disable() *lightning.ChannelDisabled {
	return &lightning.ChannelDisabled{Read: false, Write: e.code == 401 || e.code == 403 || (e.code == 404 && !e.edit)}
}

func (e *revoltStatusError) Error() string {
	return "revolt status code " + strconv.Itoa(e.code) + ": " + e.msg
}
