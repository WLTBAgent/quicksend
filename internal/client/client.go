package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type FileMeta struct {
	Name      string    `json:"name"`
	Size      int64     `json:"size"`
	UploadedAt time.Time `json:"uploaded_at"`
}

type Client struct {
	BaseURL string
	Key     string
}

func New(baseURL, key string) *Client {
	return &Client{BaseURL: strings.TrimRight(baseURL, "/"), Key: key}
}

func (c *Client) Upload(filePath string) error {
	f, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("open file: %w", err)
	}
	defer f.Close()

	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	part, err := w.CreateFormFile("file", filepath.Base(filePath))
	if err != nil {
		return fmt.Errorf("create form: %w", err)
	}
	if _, err := io.Copy(part, f); err != nil {
		return fmt.Errorf("write form: %w", err)
	}
	w.Close()

	req, err := http.NewRequest(http.MethodPost, c.BaseURL+"/upload", &body)
	if err != nil {
		return err
	}
	req.Header.Set("X-Key", c.Key)
	req.Header.Set("Content-Type", w.FormDataContentType())

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("upload failed (%d): %s", resp.StatusCode, string(b))
	}
	return nil
}

func (c *Client) List() ([]FileMeta, error) {
	req, err := http.NewRequest(http.MethodGet, c.BaseURL+"/list", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Key", c.Key)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("list failed (%d): %s", resp.StatusCode, string(b))
	}

	var files []FileMeta
	if err := json.NewDecoder(resp.Body).Decode(&files); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return files, nil
}

func (c *Client) Download(filename string, dest string) error {
	req, err := http.NewRequest(http.MethodGet, c.BaseURL+"/download/"+filename, nil)
	if err != nil {
		return err
	}
	req.Header.Set("X-Key", c.Key)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("download failed (%d): %s", resp.StatusCode, string(b))
	}

	outPath := dest
	if outPath == "" {
		outPath = filename
	}
	f, err := os.Create(outPath)
	if err != nil {
		return fmt.Errorf("create file: %w", err)
	}
	defer f.Close()

	if _, err := io.Copy(f, resp.Body); err != nil {
		return fmt.Errorf("write file: %w", err)
	}
	return nil
}
