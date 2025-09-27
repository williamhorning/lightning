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
)

// UploadFile to autumn.
func (s *Session) UploadFile(tag, name string, reader io.Reader) (string, error) {
	buf := &bytes.Buffer{}
	payload := multipart.NewWriter(buf)

	fileWriter, err := payload.CreateFormFile("file", name)
	if err != nil {
		return "", fmt.Errorf("rvapi: failed to add form file: %w", err)
	}

	if _, err = io.Copy(fileWriter, reader); err != nil {
		return "", fmt.Errorf("rvapi: failed to copy file: %w", err)
	}

	if err = payload.Close(); err != nil {
		log.Printf("rvapi: failed to close file payload: %v\n", err)
	}

	url := "https://cdn.revoltusercontent.com/" + tag

	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, url, buf)
	if err != nil {
		return "", fmt.Errorf("rvapi: failed to create request: %w", err)
	}

	req.Header["Content-Type"] = []string{payload.FormDataContentType()}
	req.Header["X-Bot-Token"] = []string{s.Token}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("revolt: failed to do request in upload: %w\n\tname: %s\n\ttag: %s", err, name, tag)
	}

	defer func() {
		if err = resp.Body.Close(); err != nil {
			log.Printf("rvapi: failed to close upload body: %v\n", err)
		}
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("rvapi: failed to read response: %w\n\tname: %s\n\ttag: %s", err, name, tag)
	}

	var response File
	if err = json.Unmarshal(body, &response); err != nil {
		return "", fmt.Errorf("rvapi: failed to unmarshal response: %w\n\tbody: %s", err, string(body))
	}

	return response.ID, nil
}
