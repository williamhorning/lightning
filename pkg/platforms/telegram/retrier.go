package telegram

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/PaulSonOfLars/gotgbot/v2"
)

type retrier struct {
	baseClient *gotgbot.BaseBotClient
}

func newRetrier() *retrier {
	return &retrier{&gotgbot.BaseBotClient{
		Client: http.Client{},
		DefaultRequestOpts: &gotgbot.RequestOpts{
			Timeout: time.Second * 10,
		},
	}}
}

func (r *retrier) RequestWithContext(
	ctx context.Context,
	token string,
	method string,
	params map[string]string,
	data map[string]gotgbot.FileReader,
	opts *gotgbot.RequestOpts,
) (json.RawMessage, error) {
	resp, err := r.baseClient.RequestWithContext(ctx, token, method, params, data, opts)
	if err == nil {
		return resp, nil
	}

	telegramError := &gotgbot.TelegramError{}
	if !errors.As(err, &telegramError) {
		return resp, err //nolint:wrapcheck // this might be used by gotgbot
	}

	if telegramError.Code != http.StatusTooManyRequests {
		return resp, err //nolint:wrapcheck // this might be used by gotgbot
	}

	time.Sleep(time.Second * time.Duration(telegramError.ResponseParams.RetryAfter))

	return r.RequestWithContext(ctx, token, method, params, data, opts)
}

func (r *retrier) GetAPIURL(opts *gotgbot.RequestOpts) string {
	return r.baseClient.GetAPIURL(opts)
}

func (r *retrier) FileURL(token string, tgFilePath string, opts *gotgbot.RequestOpts) string {
	return r.baseClient.FileURL(token, tgFilePath, opts)
}
