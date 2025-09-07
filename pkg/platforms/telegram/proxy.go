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
			slog.Warn(fmt.Errorf("telegram: failed to create request: %w", err).Error())

			return
		}

		req.Header = request.Header.Clone()

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			http.Error(writer, "Failed to fetch file from Telegram", http.StatusInternalServerError)
			slog.Warn(fmt.Errorf("telegram: failed to fetch file: %w", err).Error())

			return
		}

		maps.Copy(writer.Header(), resp.Header)
		writer.WriteHeader(resp.StatusCode)

		if _, err = io.CopyBuffer(writer, resp.Body, nil); err != nil {
			http.Error(writer, "Failed to write response", http.StatusInternalServerError)
			slog.Warn(fmt.Errorf("telegram: failed to write response: %w", err).Error())
		}

		if err = resp.Body.Close(); err != nil {
			slog.Warn(fmt.Errorf("telegram: failed to close body: %w", err).Error())
		}
	})

	addr := ":" + strconv.FormatInt(p.cfg.proxyPort, 10)

	server := &http.Server{Addr: addr, Handler: nil, ReadTimeout: defaultTimeout, WriteTimeout: defaultTimeout}

	if err := server.ListenAndServe(); err != nil {
		panic(fmt.Errorf("telegram: failed to start file proxy: %w", err))
	}

	slog.Info("telegram: file proxy available", "url", p.cfg.proxyURL, "port", p.cfg.proxyPort)
}
