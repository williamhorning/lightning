package guilded

import "strconv"

type guildedWebhookDataError struct{}

func (guildedWebhookDataError) Error() string {
	return "invalid webhook data for Guilded channel"
}

type guildedStatusError struct {
	msg  string
	code int
}

func (e guildedStatusError) Error() string {
	return strconv.Itoa(e.code) + ": " + e.msg
}

type guildedWebhookTokenNilError struct{}

func (guildedWebhookTokenNilError) Error() string {
	return "webhook token is nil"
}
