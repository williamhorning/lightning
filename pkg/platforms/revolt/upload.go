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
		slog.Error("revolt: failed to create request in upload", "error", err, "tag", tag, "name", name)

		return "", fmt.Errorf("revolt: failed to create request in upload: %w", err)
	}

	req.Header.Set("Content-Type", contentType)

	req.Header.Set("X-Bot-Token", p.token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		slog.Error("revolt: failed to do request in upload", "error", err, "tag", tag, "name", name)

		return "", fmt.Errorf("revolt: failed to do request in upload: %w", err)
	}

	defer func() {
		if err = resp.Body.Close(); err != nil {
			slog.Warn("revolt: failed to close upload body", "err", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return "", &revoltStatusError{"failed to upload file", resp.StatusCode, true}
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		slog.Error("revolt: failed to read response in upload", "error", err, "tag", tag, "name", name)

		return "", fmt.Errorf("revolt: failed to read response in upload: %w", err)
	}

	var response revoltUploadResponse
	if err = json.Unmarshal(body, &response); err != nil {
		slog.Error("revolt: failed to unmarshal response in upload", "error", err, "body", string(body),
			"tag", tag, "name", name)

		return "", fmt.Errorf("revolt: failed to unmarshal response in upload: %w", err)
	}

	return response.ID, nil
}

func createMultipartPayload(name string, reader io.Reader, tag string) (*bytes.Buffer, string, error) {
	payload := &bytes.Buffer{}
	writer := multipart.NewWriter(payload)

	fileWriter, err := writer.CreateFormFile("file", name)
	if err != nil {
		slog.Error("revolt: failed to create file field in upload", "error", err, "tag", tag, "name", name)

		return nil, "", fmt.Errorf("revolt: failed to create file field in upload: %w", err)
	}

	_, err = io.Copy(fileWriter, reader)
	if err != nil {
		slog.Error("revolt: failed to copy file in upload", "error", err, "tag", tag, "name", name)

		return nil, "", fmt.Errorf("revolt: failed to copy file in upload: %w", err)
	}

	if err = writer.Close(); err != nil {
		slog.Error("revolt: failed to close writer in upload", "error", err, "tag", tag, "name", name)

		return nil, "", fmt.Errorf("revolt: failed to close writer in upload: %w", err)
	}

	return payload, writer.FormDataContentType(), nil
}
