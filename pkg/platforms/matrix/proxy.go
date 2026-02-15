package matrix

import (
	"errors"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/id"
)

func startProxy(client *mautrix.Client, url, port string) {
	server := &http.Server{
		Addr: ":" + port, Handler: http.HandlerFunc(func(writer http.ResponseWriter, req *http.Request) {
			curl, err := id.ParseContentURI(strings.TrimPrefix(req.URL.Path, "/matrix/"))
			if err != nil {
				writer.WriteHeader(http.StatusBadRequest)

				return
			}

			resp, err := client.Download(req.Context(), curl)
			if err != nil {
				writer.WriteHeader(http.StatusInternalServerError)

				return
			}

			defer resp.Body.Close()

			if _, err = io.Copy(writer, resp.Body); err != nil {
				writer.WriteHeader(http.StatusInternalServerError)

				return
			}

			writer.Header().Set("Content-Type", resp.Header.Get("Content-Type"))
			writer.WriteHeader(http.StatusOK)
		}), ReadTimeout: 5 * time.Second, WriteTimeout: 5 * time.Second,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("matrix: error in file proxy serve: %v\n", err)
		}
	}()

	log.Printf("matrix: file proxy listening at :%s and %s\n", port, url)
}
