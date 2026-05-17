package main

import (
	"context"
	"fmt"
	"local-ai-search/internal/db"
	"local-ai-search/internal/llm"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct
type App struct {
	ctx        context.Context
	appDataDir string
	db         *db.DB
}

// NewApp creates a new App application struct
func NewApp() *App {
	// Setup app data dir
	homeDir, _ := os.UserHomeDir()
	appDataDir := filepath.Join(homeDir, ".local-ai-search")
	os.MkdirAll(appDataDir, 0755)

	// Remove old index to force clean re-index with updated skip rules
	oldDB := filepath.Join(appDataDir, "index.db")
	os.Remove(oldDB)
	os.Remove(oldDB + "-wal")
	os.Remove(oldDB + "-shm")

	database, err := db.InitDB(appDataDir)
	if err != nil {
		fmt.Printf("Failed to init db: %v\n", err)
	}

	return &App{
		appDataDir: appDataDir,
		db:         database,
	}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	
	// Scan Documents directory in background
	go func() {
		homeDir, _ := os.UserHomeDir()
		docsDir := filepath.Join(homeDir, "Documents")
		a.ScanDirectory(docsDir)
	}()
}

// ScanDirectory indexes files in a given directory
func (a *App) ScanDirectory(dirPath string) error {
	if a.db == nil {
		return fmt.Errorf("db not initialized")
	}

	runtime.EventsEmit(a.ctx, "index-start")
	
	indexedCount := 0
	filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		
		// Skip hidden directories and common dependency/build directories
		if info.IsDir() {
			name := info.Name()
			skipDirs := map[string]bool{
				"node_modules": true, "dist": true, "build": true,
				"vendor": true, "__pycache__": true, ".cache": true,
				"target": true, "bin": true, "obj": true,
				"Pods": true, "DerivedData": true, ".gradle": true,
				".pub-cache": true, ".dart_tool": true,
				"coverage": true, ".next": true, ".nuxt": true,
			}
			if strings.HasPrefix(name, ".") || skipDirs[name] {
				return filepath.SkipDir
			}
			return nil
		}

		if !info.IsDir() {
			// Skip hidden files
			if strings.HasPrefix(info.Name(), ".") {
				return nil
			}
			
			summary := fmt.Sprintf("File %s located in %s", info.Name(), filepath.Dir(path))
			
			// Deep Document Indexing
			ext := strings.ToLower(filepath.Ext(info.Name()))
			readableExts := map[string]bool{
				".txt": true, ".md": true, ".csv": true, ".json": true, 
				".go": true, ".js": true, ".ts": true, ".tsx": true, 
				".html": true, ".css": true, ".log": true,
			}
			
			if readableExts[ext] {
				content, err := os.ReadFile(path)
				if err == nil {
					// Read up to first 10KB
					contentStr := string(content)
					if len(contentStr) > 10000 {
						contentStr = contentStr[:10000]
					}
					// Only use content if it's mostly valid UTF-8 (simple check by replacing invalid chars)
					contentStr = strings.ToValidUTF8(contentStr, " ")
					summary = contentStr
				}
			}
			
			// Index it
			a.db.AddDocument(path, info.Name(), summary, []string{"Auto-Indexed"}, info.ModTime().Unix())
			indexedCount++

			// Emit progress every 5 files or on the first file
			if indexedCount%5 == 0 || indexedCount == 1 {
				runtime.EventsEmit(a.ctx, "index-progress", map[string]interface{}{
					"count":   indexedCount,
					"current": info.Name(),
				})
			}
		}
		return nil
	})
	
	runtime.EventsEmit(a.ctx, "index-complete", indexedCount)
	return nil
}

// ClearDatabase empties the database completely
func (a *App) ClearDatabase() error {
	if a.db == nil {
		return fmt.Errorf("database not initialized")
	}
	return a.db.ClearAll()
}

// Search performs a query across the indexed documents
func (a *App) Search(query string) ([]db.SearchResult, error) {
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	return a.db.Search(query)
}

// AISearch performs an AI-powered reranking of search results
func (a *App) AISearch(query string) ([]llm.AISearchResult, error) {
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	// Get raw results from FTS
	rawResults, err := a.db.Search(query)
	if err != nil {
		return nil, err
	}

	// Convert to AISearchResult
	var aiResults []llm.AISearchResult
	for _, r := range rawResults {
		aiResults = append(aiResults, llm.AISearchResult{
			Path:     r.Path,
			Filename: r.Filename,
			Summary:  r.Summary,
		})
	}

	// Run the smart reranker (instant, no LLM needed)
	reranked := llm.SmartRerank(query, aiResults)

	return reranked, nil
}

// OpenFile opens a file in the default OS application
func (a *App) OpenFile(path string) error {
	var cmd *exec.Cmd
	switch os := runtime.Environment(a.ctx).Platform; os {
	case "darwin":
		cmd = exec.Command("open", path)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", path)
	default: // linux
		cmd = exec.Command("xdg-open", path)
	}
	return cmd.Start()
}

// OpenFolder opens the directory containing the file in the default file manager
func (a *App) OpenFolder(path string) error {
	var cmd *exec.Cmd
	switch os := runtime.Environment(a.ctx).Platform; os {
	case "darwin":
		cmd = exec.Command("open", "-R", path)
	case "windows":
		cmd = exec.Command("explorer", "/select,"+filepath.Clean(path))
	default: // linux
		cmd = exec.Command("xdg-open", filepath.Dir(path))
	}
	return cmd.Start()
}

// DownloadLLM starts downloading the LLM and emits progress events
func (a *App) DownloadLLM() (string, error) {
	runtime.EventsEmit(a.ctx, "llm-download-start", "Starting download...")
	
	modelPath, err := llm.DownloadModel(a.ctx, a.appDataDir, func(p llm.DownloadProgress) {
		runtime.EventsEmit(a.ctx, "llm-download-progress", p)
	})

	if err != nil {
		runtime.EventsEmit(a.ctx, "llm-download-error", err.Error())
		return "", err
	}

	runtime.EventsEmit(a.ctx, "llm-download-complete", modelPath)
	return modelPath, nil
}

// GetSettings gets settings (mock for now)
func (a *App) GetSettings() map[string]interface{} {
	return map[string]interface{}{
		"model_path": filepath.Join(a.appDataDir, "models", llm.ModelName),
		"platform":   runtime.Environment(a.ctx).Platform,
	}
}

