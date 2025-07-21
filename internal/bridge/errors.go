package bridge

import (
	"fmt"
)

type disableChannelError struct {
	BridgeID  string
	ChannelID string
}

func (e disableChannelError) Error() string {
	return "disabling channel " + e.ChannelID + " in bridge " + e.BridgeID
}

type unsupportedTypeError struct {
	Type any
}

func (e unsupportedTypeError) Error() string {
	return "unsupported type: " + fmt.Sprint(e.Type)
}
