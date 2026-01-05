package matrix

import (
	"errors"
	"fmt"

	"codeberg.org/jersey/lightning/pkg/lightning"
	"maunium.net/go/mautrix"
)

type matrixError struct {
	method string
	code   string
}

func (e matrixError) Disable() *lightning.ChannelDisabled {
	return &lightning.ChannelDisabled{Read: false, Write: e.code == "M_FORBIDDEN" || e.code == "M_UNAUTHORIZED"}
}

func (e matrixError) Error() string {
	return "failed to " + e.method + " message: "
}

func handleError(err error, method string) error {
	var httpErr *mautrix.HTTPError
	if !errors.As(err, &httpErr) || httpErr.RespError == nil {
		return fmt.Errorf("failed to %s message: %w", method, err)
	}

	return &matrixError{method, httpErr.RespError.ErrCode}
}
