package stoat

import (
	"strconv"

	"github.com/williamhorning/lightning/internal/v2/stoat"
	"github.com/williamhorning/lightning/pkg/lightning"
)

type stoatDMError struct{}

func (*stoatDMError) Error() string {
	return "Failed to DM you! Please check that the bot can DM you."
}

type stoatPermissionsError struct {
	permissions stoat.Permission
	expected    stoat.Permission
}

func (*stoatPermissionsError) Disable() *lightning.ChannelDisabled {
	return &lightning.ChannelDisabled{Read: false, Write: true}
}

func (e *stoatPermissionsError) Error() string {
	err := "Missing the following permissions, please ensure these are granted: `"

	for permission, name := range stoat.PermissionNames {
		if (e.expected&permission == permission) && (e.permissions&permission != permission) {
			err += name + " "
		}
	}

	return err + strconv.FormatUint(uint64(e.permissions), 10) + "&" + strconv.FormatUint(uint64(e.expected), 10) + "`"
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
