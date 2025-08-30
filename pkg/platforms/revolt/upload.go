package revolt

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
)

func (p *revoltPlugin) uploadFile(tag, name string, reader io.Reader) (string, error) {
	url := "https://cdn.revoltusercontent.com/" + tag

	payload, contentType, err := createMultipartPayload(name, reader, tag)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, url, payload)
	if err != nil {
		return "", fmt.Errorf("revolt: failed to create request in upload: %w\n\tname: %s\n\ttag: %s", err, name, tag)
	}

	req.Header.Set("Content-Type", contentType)

	req.Header.Set("X-Bot-Token", p.token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("revolt: failed to do request in upload: %w\n\tname: %s\n\ttag: %s", err, name, tag)
	}

	defer func() {
		if err = resp.Body.Close(); err != nil {
			slog.Warn(fmt.Errorf("revolt: failed to close upload body: %w", err).Error())
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return "", &revoltStatusError{"failed to upload file", resp.StatusCode, true}
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("revolt: failed to read response in upload: %w\n\tname: %s\n\ttag: %s", err, name, tag)
	}

	var response revoltUploadResponse
	if err = json.Unmarshal(body, &response); err != nil {
		return "", fmt.Errorf("revolt: failed to unmarshal file in upload: %w\n\tname: %s\n\ttag: %s", err, name, tag)
	}

	return response.ID, nil
}

func createMultipartPayload(name string, reader io.Reader, tag string) (*bytes.Buffer, string, error) {
	payload := &bytes.Buffer{}
	writer := multipart.NewWriter(payload)

	fileWriter, err := writer.CreateFormFile("file", name)
	if err != nil {
		return nil, "", fmt.Errorf("revolt: failed to create file in upload: %w\n\tname: %s\n\ttag: %s", err, name, tag)
	}

	_, err = io.Copy(fileWriter, reader)
	if err != nil {
		return nil, "", fmt.Errorf("revolt: failed to copy file in upload: %w\n\tname: %s\n\ttag: %s", err, name, tag)
	}

	if err = writer.Close(); err != nil {
		return nil, "", fmt.Errorf("revolt: failed to close upload writer: %w\n\tname: %s\n\ttag: %s", err, name, tag)
	}

	return payload, writer.FormDataContentType(), nil
}
