package revolt

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"strconv"

	"github.com/williamhorning/lightning/pkg/lightning"
)

func uploadFile(token, tag, name string, reader io.Reader) (string, error) {
	url := "https://cdn.revoltusercontent.com/" + tag

	payload := &bytes.Buffer{}
	writer := multipart.NewWriter(payload)

	fileWriter, err := writer.CreateFormFile("file", name)
	if err != nil {
		return "", lightning.LogError(err, "revolt: failed to make file field in upload", nil, nil)
	}

	_, err = io.Copy(fileWriter, reader)
	if err != nil {
		return "", lightning.LogError(err, "revolt: failed to copy file in upload", nil, nil)
	}

	if err = writer.Close(); err != nil {
		return "", lightning.LogError(err, "revolt: failed to close writer in upload", nil, nil)
	}

	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, url, payload)
	if err != nil {
		return "", lightning.LogError(err, "revolt: failed to make request in upload", nil, nil)
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("X-Bot-Token", token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", lightning.LogError(err, "revolt: failed to do request in upload", nil, nil)
	}

	defer func() {
		if err = resp.Body.Close(); err != nil {
			slog.Warn("revolt: failed to close upload body", "err", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return "", lightning.LogError(
			revoltStatusError{"failed to upload file", resp.StatusCode},
			"revolt: unexpected status code "+strconv.Itoa(resp.StatusCode),
			nil,
			nil,
		)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", lightning.LogError(err, "revolt: failed to read response in upload", nil, nil)
	}

	var response revoltUploadResponse

	if err = json.Unmarshal(body, &response); err != nil {
		return "", lightning.LogError(err, "revolt: failed to unmarshal response in upload", nil, nil)
	}

	return response.ID, nil
}
