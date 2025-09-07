package telegram

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/PaulSonOfLars/gotgbot/v2"
)

const defaultTimeout = gotgbot.DefaultTimeout * 2

type retrier struct {
	baseClient *gotgbot.BaseBotClient
}

func newRetrier() *retrier {
	return &retrier{&gotgbot.BaseBotClient{DefaultRequestOpts: &gotgbot.RequestOpts{Timeout: defaultTimeout}}}
}

func (r *retrier) RequestWithContext(
	ctx context.Context,
	token, method string,
	params map[string]string,
	data map[string]gotgbot.FileReader,
	opts *gotgbot.RequestOpts,
) (json.RawMessage, error) {
	resp, err := r.baseClient.RequestWithContext(ctx, token, method, params, data, opts)
	if err == nil {
		return resp, nil
	}

	urlError := &url.Error{}
	if errors.As(err, &urlError) {
		urlError.URL = strings.ReplaceAll(urlError.URL, token, "")

		return resp, urlError
	}

	telegramError := &gotgbot.TelegramError{}
	if !errors.As(err, &telegramError) || telegramError.Code != 429 {
		return resp, fmt.Errorf("error making request in retrier: %w", err)
	}

	time.Sleep(time.Second * time.Duration(telegramError.ResponseParams.RetryAfter))

	return r.RequestWithContext(ctx, token, method, params, data, opts)
}

func (r *retrier) GetAPIURL(opts *gotgbot.RequestOpts) string {
	return r.baseClient.GetAPIURL(opts)
}

func (r *retrier) FileURL(token, tgFilePath string, opts *gotgbot.RequestOpts) string {
	return r.baseClient.FileURL(token, tgFilePath, opts)
}
