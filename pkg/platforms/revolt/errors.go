package revolt

import (
	"errors"
	"strconv"

	"github.com/williamhorning/lightning/pkg/lightning"
)

type revoltPermissionsError struct{}

func (revoltPermissionsError) Error() string {
	return "insufficient permissions in Revolt channel, please check them"
}

type revoltStatusError struct {
	msg  string
	code int
}

func (e revoltStatusError) Error() string {
	return strconv.Itoa(e.code) + ": " + e.msg
}

func getRevoltError(err error, extra map[string]any, message string, edit bool) error {
	if errors.Is(err, revoltPermissionsError{}) {
		return lightning.LogError(err, err.Error(), nil, &lightning.ChannelDisabled{Read: false, Write: true})
	}

	var revoltErr revoltStatusError
	if errors.As(err, &revoltErr) {
		switch revoltErr.code {
		case 403:
			return lightning.LogError(
				err,
				"insufficient permissions, please check them",
				extra,
				&lightning.ChannelDisabled{Read: false, Write: true},
			)
		case 404:
			if edit {
				return nil
			}

			return lightning.LogError(err, "revolt: resource not found",
				extra, &lightning.ChannelDisabled{Read: false, Write: true})
		default:
			return lightning.LogError(err, message, extra, nil)
		}
	}

	return lightning.LogError(err, message, extra, nil)
}
