package matrix

import (
	"errors"
	"fmt"
	"log"
	"net/http"

	"github.com/williamhorning/lightning/pkg/lightning"
	"maunium.net/go/mautrix"
)

type matrixError struct {
	err          error
	msg          string
	disableWrite bool
}

// Disable implements lightning.ChannelDisabler.
func (e matrixError) Disable() *lightning.ChannelDisabled {
	return &lightning.ChannelDisabled{Write: e.disableWrite}
}

func (e matrixError) Error() string {
	return e.msg
}

func (e matrixError) Unwrap() error {
	return e.err
}

func handleError(err error, msg string, extra map[string]any) error {
	log.Printf("matrix: %s: %v\n\textra: %#+v\n", msg, err, extra)

	var httpErr *mautrix.HTTPError
	if !errors.As(err, &httpErr) {
		return fmt.Errorf("matrix error: %w", err)
	}

	extra["err_msg"] = httpErr.Message

	extra["status_code"] = httpErr.Response.StatusCode
	if httpErr.RespError == nil {
		return fmt.Errorf("matrix error: %w", err)
	}

	extra["err_code"] = httpErr.RespError.ErrCode

	disable := false

	switch httpErr.RespError.StatusCode {
	case http.StatusForbidden, http.StatusNotFound:
		disable = true
	default:
	}

	return &matrixError{err, msg, disable}
}
