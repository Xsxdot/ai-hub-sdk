// media_upload.go implements public media upload for ai-hub SDK.
//
// Responsibilities:
//   - Send multipart files to POST /v1/media with X-API-Key
//   - Decode the result envelope into an ai-hub-issued ossKey response
//
// Boundaries:
//   - Does not inspect file content or guess MIME type
//   - Does not retry uploads, because uploads are not guaranteed idempotent
package aihubsdk

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"

	"github.com/xsxdot/ai-hub-sdk/dto"
)

// UploadMediaKind is the public upload media category.
type UploadMediaKind string

const (
	// UploadMediaKindImage stores an uploaded image file.
	UploadMediaKindImage UploadMediaKind = "image"
	// UploadMediaKindVideo stores an uploaded video file.
	UploadMediaKindVideo UploadMediaKind = "video"
	// UploadMediaKindAudio stores an uploaded audio file.
	UploadMediaKindAudio UploadMediaKind = "audio"
)

// UploadMedia uploads one file to ai-hub and returns an ai-hub-issued ossKey.
func (c *Client) UploadMedia(ctx context.Context, kind UploadMediaKind, filename string, r io.Reader) (*dto.MediaUploadResult, error) {
	if r == nil {
		return nil, fmt.Errorf("upload media reader is nil")
	}
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if err := writer.WriteField("kind", string(kind)); err != nil {
		return nil, fmt.Errorf("write kind field: %w", err)
	}
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return nil, fmt.Errorf("create file field: %w", err)
	}
	if _, err := io.Copy(part, r); err != nil {
		return nil, fmt.Errorf("copy upload file: %w", err)
	}
	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("close multipart writer: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/media", &body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-API-Key", c.apiKey)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	setTraceHeaders(req, ctx)

	var out dto.MediaUploadResult
	if err := c.doRequest(req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
