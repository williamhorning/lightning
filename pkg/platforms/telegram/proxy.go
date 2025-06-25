package telegram

import (
	"io"
	"maps"
	"net/http"
	"strconv"
	"strings"

	"github.com/williamhorning/lightning/pkg/lightning"
)

func (p *telegramPlugin) startProxy() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/telegram")
		url := p.telegram.FileURL(p.telegram.Token, path, nil)
		req, err := http.NewRequestWithContext(r.Context(), r.Method, url, nil)
		if err != nil {
			http.Error(w, "Failed to create request", http.StatusInternalServerError)
			lightning.Log.Error().Str("plugin", "telegram").Err(err).Msg("Failed to create request for Telegram file proxy")
			return
		}

		req.Header = r.Header.Clone()
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			http.Error(w, "Failed to fetch file from Telegram", http.StatusInternalServerError)
			lightning.Log.Error().Str("plugin", "telegram").Err(err).Msg("Failed to fetch file from Telegram")
			return
		}

		defer resp.Body.Close()
		maps.Copy(w.Header(), resp.Header)
		w.WriteHeader(resp.StatusCode)
		if _, err := io.CopyBuffer(w, resp.Body, make([]byte, 64*1024)); err != nil {
			http.Error(w, "Failed to write response", http.StatusInternalServerError)
			lightning.Log.Error().Str("plugin", "telegram").Err(err).Msg("Failed to write response from Telegram file proxy")
			return
		}
	})

	if err := http.ListenAndServe("0.0.0.0:"+strconv.FormatInt(p.proxyPort, 10), nil); err != nil {
		lightning.Log.Panic().Str("plugin", "telegram").Err(err).Msg("Failed to start Telegram file proxy")
	}

	lightning.Log.Info().
		Str("plugin", "telegram").
		Int64("port", p.proxyPort).
		Str("url", p.proxyURL).
		Msg("Telegram file proxy available")
}
