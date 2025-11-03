package stoat

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"time"

	"github.com/williamhorning/lightning/internal/v2/workaround"
)

// UploadFile uploads a file to Autumn.
func (s *Session) UploadFile(tag, srcURL, filename string) (*CDNFile, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)

	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, srcURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create download request: %w", err)
	}

	resp, err := workaround.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to download file: %w", err)
	}

	defer func() {
		if err = resp.Body.Close(); err != nil {
			log.Printf("internal/stoat: failed to close body: %v\n", err)
		}
	}()

	var buf bytes.Buffer

	writer := multipart.NewWriter(&buf)

	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return nil, fmt.Errorf("failed to create form file: %w", err)
	}

	if _, err = io.Copy(part, resp.Body); err != nil {
		return nil, fmt.Errorf("failed to copy downloaded data: %w", err)
	}

	if err = writer.Close(); err != nil {
		return nil, fmt.Errorf("failed to finalize multipart payload: %w", err)
	}

	base := "https://cdn.stoatusercontent.com/"

	body, _, err := s.Fetch(http.MethodPost, "/"+tag, &buf, &base,
		map[string][]string{"Content-Type": {writer.FormDataContentType()}})
	if err != nil {
		return nil, fmt.Errorf("upload failed: %w", err)
	}

	defer func() {
		if err = body.Close(); err != nil {
			log.Printf("internal/stoat: failed to close body: %v\n", err)
		}
	}()

	var uploaded CDNFile
	if err := json.NewDecoder(body).Decode(&uploaded); err != nil {
		return nil, fmt.Errorf("failed to decode upload response: %w", err)
	}

	return &uploaded, nil
}
