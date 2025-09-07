package revolt

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/williamhorning/lightning/pkg/lightning"
)

type revoltPermissionsError struct {
	msg         string
	permissions uint
	expected    uint
}

func (revoltPermissionsError) Disable() *lightning.ChannelDisabled {
	return &lightning.ChannelDisabled{Read: false, Write: true}
}

func (e revoltPermissionsError) Error() string {
	return "insufficient permissions in Revolt (have " +
		strconv.FormatUint(uint64(e.permissions), 10) + ", want " +
		strconv.FormatUint(uint64(e.expected), 10) + ")" + e.msg + " channel, please check them"
}

type revoltStatusError struct {
	msg  string
	code int
	edit bool
}

func (e revoltStatusError) Disable() *lightning.ChannelDisabled {
	if e.code == 403 || (e.code == 404 && !e.edit) {
		return &lightning.ChannelDisabled{Read: false, Write: true}
	}

	return &lightning.ChannelDisabled{Read: false, Write: false}
}

func (e revoltStatusError) Error() string {
	return strconv.Itoa(e.code) + ": " + e.msg
}

func getRevoltError(err error, extra map[string]any, message string) error {
	if errors.Is(err, revoltPermissionsError{}) {
		return err
	}

	if errors.Is(err, revoltStatusError{}) {
		return fmt.Errorf("revolt: status error: %w\n\textra: %#+v", err, extra)
	}

	return fmt.Errorf("revolt: %s: %w\n\textra: %#+v", message, err, extra)
}
