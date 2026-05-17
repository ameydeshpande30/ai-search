package llm

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

// We'll use a very small, capable model for embeddings and tagging.
// For example, an all-MiniLM-L6-v2 GGUF or TinyLlama.
// TinyLlama 1.1B 4-bit is around 630MB.
const (
	ModelURL  = "https://huggingface.co/TheBloke/TinyLlama-1.1B-Chat-v1.0-GGUF/resolve/main/tinyllama-1.1b-chat-v1.0.Q4_K_M.gguf"
	ModelName = "tinyllama-1.1b-chat-v1.0.Q4_K_M.gguf"
)

type DownloadProgress struct {
	Total      uint64
	Downloaded uint64
	Percentage int
}

// DownloadModel downloads the small LLM to the app's local data directory if it doesn't exist
func DownloadModel(ctx context.Context, appDataDir string, progressCallback func(DownloadProgress)) (string, error) {
	modelsDir := filepath.Join(appDataDir, "models")
	if err := os.MkdirAll(modelsDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create models directory: %w", err)
	}

	modelPath := filepath.Join(modelsDir, ModelName)

	// Check if already exists
	if info, err := os.Stat(modelPath); err == nil && info.Size() > 0 {
		return modelPath, nil // Already downloaded
	}

	// Create temporary file
	tmpPath := modelPath + ".tmp"
	out, err := os.Create(tmpPath)
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	defer out.Close()

	resp, err := http.Get(ModelURL)
	if err != nil {
		return "", fmt.Errorf("failed to download model: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("bad status: %s", resp.Status)
	}

	totalSize := uint64(resp.ContentLength)
	
	// Create a custom reader to track progress
	counter := &writeCounter{
		Total:      totalSize,
		Callback:   progressCallback,
	}

	if _, err := io.Copy(out, io.TeeReader(resp.Body, counter)); err != nil {
		return "", fmt.Errorf("failed to save model: %w", err)
	}

	out.Close() // Close before renaming

	// Rename temp file to actual file
	if err := os.Rename(tmpPath, modelPath); err != nil {
		return "", fmt.Errorf("failed to rename temp file: %w", err)
	}

	return modelPath, nil
}

type writeCounter struct {
	Total      uint64
	Downloaded uint64
	Callback   func(DownloadProgress)
	lastPercent int
}

func (wc *writeCounter) Write(p []byte) (int, error) {
	n := len(p)
	wc.Downloaded += uint64(n)
	
	if wc.Total > 0 && wc.Callback != nil {
		percent := int((float64(wc.Downloaded) / float64(wc.Total)) * 100)
		if percent > wc.lastPercent {
			wc.lastPercent = percent
			wc.Callback(DownloadProgress{
				Total:      wc.Total,
				Downloaded: wc.Downloaded,
				Percentage: percent,
			})
		}
	}
	
	return n, nil
}
