package telegram

import (
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
)

func startProxy(cfg map[string]string) error {
	listener, err := net.Listen("tcp", ":"+cfg["proxy_port"])
	if err != nil {
		return fmt.Errorf("failed to start file proxy listener: %w", err)
	}

	server := &http.Server{
		Addr: ":" + cfg["proxy_port"], Handler: &httputil.ReverseProxy{Director: func(req *http.Request) {
			req.URL = &url.URL{
				Scheme: "https", Host: "api.telegram.org",
				Path: "/file/bot" + cfg["token"] + "/" + strings.TrimPrefix(req.URL.Path, "/telegram"),
			}
			req.Host = "api.telegram.org"
		}}, ReadTimeout: defaultTimeout, WriteTimeout: defaultTimeout,
	}

	go func() {
		if err := server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("telegram: error in file proxy serve: %v\n", err)
		}
	}()

	log.Printf("telegram: file proxy listening at :%s and %s\n", cfg["proxy_port"], cfg["proxy_url"])

	return nil
}
