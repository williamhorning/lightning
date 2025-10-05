package telegram

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
)

func startProxy(cfg map[string]string) {
	proxy := &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			req.URL = &url.URL{
				Scheme: "https",
				Host:   "api.telegram.org",
				Path:   "/file/bot" + cfg["token"] + "/" + strings.TrimPrefix(req.URL.Path, "/telegram"),
			}
			req.Host = "api.telegram.org"
		},
	}

	server := &http.Server{
		Addr: ":" + cfg["proxy_port"], Handler: proxy, ReadTimeout: defaultTimeout, WriteTimeout: defaultTimeout,
	}

	if err := server.ListenAndServe(); err != nil {
		panic(fmt.Errorf("telegram: failed to start file proxy: %w", err))
	}

	log.Printf("telegram file proxy (port %s) available at %s\n", cfg["proxy_port"], cfg["proxy_url"])
}
