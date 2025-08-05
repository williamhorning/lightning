package matrix

import (
	"errors"
	"net/http"

	"github.com/williamhorning/lightning/pkg/lightning"
	"maunium.net/go/mautrix"
)

func handleError(err error, msg string, extra map[string]any) error {
	var httpErr *mautrix.HTTPError
	if !errors.As(err, &httpErr) {
		return lightning.LogError(err, msg, extra, nil)
	}

	extra["err_msg"] = httpErr.Message

	extra["status_code"] = httpErr.Response.StatusCode
	if httpErr.RespError == nil {
		return lightning.LogError(err, msg, extra, nil)
	}

	extra["err_code"] = httpErr.RespError.ErrCode

	disable := &lightning.ChannelDisabled{}

	switch httpErr.RespError.StatusCode {
	case http.StatusForbidden, http.StatusNotFound:
		disable.Write = true
	}

	return lightning.LogError(err, msg, extra, disable)
}
