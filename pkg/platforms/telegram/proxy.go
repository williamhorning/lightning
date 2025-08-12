package telegram

import (
	"fmt"
	"io"
	"log/slog"
	"maps"
	"net/http"
	"strconv"
	"strings"
)

func (p *telegramPlugin) startProxy() {
	http.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
		path := strings.TrimPrefix(request.URL.Path, "/telegram")
		url := p.telegram.FileURL(p.telegram.Token, path, nil)

		req, err := http.NewRequestWithContext(request.Context(), request.Method, url, nil)
		if err != nil {
			http.Error(writer, "Failed to create request", http.StatusInternalServerError)
			slog.Warn("telegram: failed to create request", "err", err)

			return
		}

		req.Header = request.Header.Clone()

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			http.Error(writer, "Failed to fetch file from Telegram", http.StatusInternalServerError)
			slog.Warn("telegram: failed to fetch file", "err", err)

			return
		}

		maps.Copy(writer.Header(), resp.Header)
		writer.WriteHeader(resp.StatusCode)

		if _, err = io.CopyBuffer(writer, resp.Body, nil); err != nil {
			http.Error(writer, "Failed to write response", http.StatusInternalServerError)
			slog.Warn("telegram: failed to write resp", "err", err)
		}

		if err = resp.Body.Close(); err != nil {
			slog.Warn("telegram: failed to close body", "err", err)
		}
	})

	//nolint:gosec // this doesn't really matter right now
	if err := http.ListenAndServe("0.0.0.0:"+strconv.FormatInt(p.cfg.proxyPort, 10), nil); err != nil {
		panic(fmt.Errorf("telegram: failed to start file proxy: %w", err))
	}

	slog.Info("telegram: file proxy available", "url", p.cfg.proxyURL, "port", p.cfg.proxyPort)
}
