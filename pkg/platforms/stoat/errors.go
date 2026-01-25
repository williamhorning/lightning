package stoat

import (
	"fmt"
	"strconv"
	"strings"

	"codeberg.org/jersey/lightning/pkg/lightning"
)

type stError struct {
	ErrorType string `json:"type"`
	Location  string `json:"location"`
	ErrorInfo string `json:"error"`
	data      any
}

func (e *stError) Disable() *lightning.ChannelDisabled {
	switch e.ErrorType {
	case "UnknownChannel":
		return &lightning.ChannelDisabled{Read: true, Write: true}
	case "MissingPermission", "MissingUserPermission", "NotElevated", "NotPrivileged", "NotOwner",
		"CannotGiveMissingPermissions", "Banned", "Blocked", "BlockedByOther":
		return &lightning.ChannelDisabled{Read: false, Write: true}
	default:
		return &lightning.ChannelDisabled{Read: false, Write: false}
	}
}

func (e *stError) Error() string {
	return e.ErrorType + " ( https://github.com/stoatchat/stoatchat/blob/main/" +
		(strings.Replace(e.Location, ":", "#L", 1)) + " ) with info: " + e.ErrorInfo + ":" + fmt.Sprintf("%#v", e.data)
}

type stoatPermissionsError struct {
	have stPermission
	want stPermission
	who  string
}

func (*stoatPermissionsError) Disable() *lightning.ChannelDisabled {
	return &lightning.ChannelDisabled{Read: false, Write: true}
}

func (e *stoatPermissionsError) Error() string {
	err := e.who + " is missing the following permissions: "

	for permission, name := range stPermissionNames {
		if e.want&permission == permission && e.have&permission != permission {
			err += "`" + name + "` "
		}
	}

	return err + "`" + strconv.FormatUint(uint64(e.have), 10) + "&" + strconv.FormatUint(uint64(e.want), 10) + "`"
}
