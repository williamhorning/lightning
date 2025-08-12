package revolt

import (
	"errors"
	"fmt"
	"log/slog"
	"strconv"

	"github.com/williamhorning/lightning/pkg/lightning"
)

type revoltPermissionsError struct {
	msg string
}

// Disable implements the lightning.ChannelDisabler interface for revoltPermissionsError.
func (revoltPermissionsError) Disable() *lightning.ChannelDisabled {
	return &lightning.ChannelDisabled{Read: false, Write: true}
}

func (e revoltPermissionsError) Error() string {
	return "insufficient permissions in Revolt " + e.msg + " channel, please check them"
}

type revoltStatusError struct {
	msg            string
	code           int
	disableDisable bool
}

// Disable implements the lightning.ChannelDisabler interface for revoltStatusError.
func (e revoltStatusError) Disable() *lightning.ChannelDisabled {
	return &lightning.ChannelDisabled{Read: false, Write: e.code == 403 || (e.code == 404 && !e.disableDisable)}
}

func (e revoltStatusError) Error() string {
	return strconv.Itoa(e.code) + ": " + e.msg
}

func getRevoltError(err error, extra map[string]any, message string) error {
	if errors.Is(err, revoltPermissionsError{}) {
		slog.Error("revolt: insufficient permissions", "error", err, "extra", extra)

		return err
	}

	if errors.Is(err, revoltStatusError{}) {
		slog.Error("revolt: status error", "error", err, "extra", extra)

		return err
	}

	slog.Error("revolt: error", "error", err, "message", message, "extra", extra)

	return fmt.Errorf("revolt: %s: %w", message, err)
}
