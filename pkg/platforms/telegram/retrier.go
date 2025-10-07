package telegram

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/williamhorning/lightning/pkg/lightning"
)

const defaultTimeout = gotgbot.DefaultTimeout * 2

type telegramAPIError struct {
	err  error
	code int
}

func (e *telegramAPIError) Disable() *lightning.ChannelDisabled {
	return &lightning.ChannelDisabled{Read: false, Write: e.code == 401 || e.code == 403}
}

func (e *telegramAPIError) Error() string {
	return "error making telegram request (" + strconv.FormatInt(int64(e.code), 10) + "): " + e.err.Error()
}

func (e *telegramAPIError) Unwrap() error {
	return e.err
}

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
		return resp, &telegramAPIError{telegramError, telegramError.Code}
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
