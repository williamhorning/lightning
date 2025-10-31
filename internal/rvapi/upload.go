package rvapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"

	"github.com/williamhorning/lightning/internal/workaround"
)

// UploadFile to Autumn.
func (s *Session) UploadFile(tag, name string, reader io.Reader) (*CDNFile, error) {
	buf := &bytes.Buffer{}
	payload := multipart.NewWriter(buf)

	fileWriter, err := payload.CreateFormFile("file", name)
	if err != nil {
		return nil, fmt.Errorf("rvapi: failed to add form file: %w", err)
	}

	if _, err = io.Copy(fileWriter, reader); err != nil {
		return nil, fmt.Errorf("rvapi: failed to copy file: %w", err)
	}

	if err = payload.Close(); err != nil {
		log.Printf("rvapi: failed to close file payload: %v\n", err)
	}

	url := "https://cdn.stoatusercontent.com/" + tag

	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, url, buf)
	if err != nil {
		return nil, fmt.Errorf("rvapi: failed to create request: %w", err)
	}

	req.Header["Content-Type"] = []string{payload.FormDataContentType()}
	req.Header["X-Bot-Token"] = []string{s.Token}

	resp, err := workaround.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("stoat: failed to do request in upload: %w\n\tname: %s\n\ttag: %s", err, name, tag)
	}

	defer func() {
		if err = resp.Body.Close(); err != nil {
			log.Printf("rvapi: failed to close upload body: %v\n", err)
		}
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("rvapi: failed to read response: %w\n\tname: %s\n\ttag: %s", err, name, tag)
	}

	var response CDNFile
	if err = json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("rvapi: failed to unmarshal response: %w\n\tbody: %s", err, string(body))
	}

	return &response, nil
}
