package matrix

import (
	"errors"
	"fmt"

	"github.com/williamhorning/lightning/pkg/lightning"
	"maunium.net/go/mautrix"
)

type matrixError struct {
	msg  string
	code int
}

func (e matrixError) Disable() *lightning.ChannelDisabled {
	return &lightning.ChannelDisabled{Read: false, Write: e.code == 403 || e.code == 404}
}

func (e matrixError) Error() string {
	return e.msg
}

func handleError(err error, msg string) error {
	var httpErr *mautrix.HTTPError
	if !errors.As(err, &httpErr) || httpErr.RespError == nil {
		return fmt.Errorf("matrix error: %w", err)
	}

	return &matrixError{msg, httpErr.RespError.StatusCode}
}
