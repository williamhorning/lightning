package revolt

import (
	"strconv"
	"strings"

	"github.com/williamhorning/lightning/pkg/lightning"
)

func extractStatusAndBody(err error) (int, string) {
	msg := err.Error()

	if !strings.HasPrefix(msg, "bad status code ") {
		return 0, ""
	}

	msg = msg[16:]
	statusCode, _ := strconv.Atoi(msg[:3])
	return statusCode, msg[5:]
}

func getRevoltError(err error, extra map[string]any, message string, edit bool) error {
	statusCode, body := extractStatusAndBody(err)

	extra["status_code"] = statusCode
	extra["body"] = body

	if statusCode == 403 {
		return lightning.LogError(err, "insufficient permissions, please check them", extra, &lightning.ChannelDisabled{Read: false, Write: true})
	} else if statusCode == 404 && edit {
		return nil
	} else if statusCode == 404 {
		return lightning.LogError(err, "resource not found", extra, &lightning.ChannelDisabled{Read: false, Write: true})
	} else {
		return lightning.LogError(err, message, extra, nil)
	}
}
