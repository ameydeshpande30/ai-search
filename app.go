package main

import (
	"context"
	"embed"
	"fmt"
	"io"
	"io/fs"
	"local-ai-search/internal/db"
	"local-ai-search/internal/llm"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct
type App struct {
	ctx        context.Context
	appDataDir string
	db         *db.DB
	modelFS    embed.FS
}

// NewApp creates a new App application struct
func NewApp(modelFS embed.FS) *App {
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
		modelFS:    modelFS,
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

	// 1. Try running GGUF Text-to-SQL
	platform := runtime.Environment(a.ctx).Platform
	cliName := "llama-cli"
	if platform == "windows" {
		cliName = "llama-cli.exe"
	}
	cliPath := filepath.Join(a.appDataDir, "bin", cliName)
	modelPath := filepath.Join(a.appDataDir, "models", llm.ModelName)

	var results []db.SearchResult
	var err error
	textToSqlSuccess := false

	// Check if both model and cli binary exist
	if _, errModel := os.Stat(modelPath); errModel == nil {
		if _, errCli := os.Stat(cliPath); errCli == nil {
			// Build Prompt
			nowSecs := time.Now().Unix()
			prompt := llm.BuildSQLPrompt(query, nowSecs)

			// Execute llama-cli with GPU offloading enabled
			cmd := exec.Command(cliPath,
				"-m", modelPath,
				"-p", prompt,
				"-n", "64",
				"--temp", "0.1",
				"-ngl", "99",
				"--log-disable",
			)

			outputBytes, errRun := cmd.Output()
			if errRun == nil {
				outputStr := string(outputBytes)
				// Extract generated assistant response after '<|assistant|>'
				idx := strings.LastIndex(outputStr, "<|assistant|>")
				var generatedSQL string
				if idx != -1 {
					generatedSQL = outputStr[idx+len("<|assistant|>"):]
				} else {
					generatedSQL = outputStr
				}
				
				// Re-extract since the tag is exactly "<|assistant|>\n"
				idxAssistantNL := strings.LastIndex(outputStr, "<|assistant|>\n")
				if idxAssistantNL != -1 {
					generatedSQL = outputStr[idxAssistantNL+len("<|assistant|>\n"):]
				}

				sanitizedSQL, errSanitize := llm.SanitizeSQLQuery(generatedSQL)
				if errSanitize == nil {
					// Query SQLite
					results, err = a.db.QueryRaw(sanitizedSQL)
					if err == nil {
						textToSqlSuccess = true
					}
				} else {
					fmt.Printf("SQL Sanitization failed: %v, raw output: %s\n", errSanitize, generatedSQL)
				}
			} else {
				fmt.Printf("llama-cli execution failed: %v\n", errRun)
			}
		}
	}

	// 2. Fallback to standard FTS search if Text-to-SQL failed or wasn't run
	if !textToSqlSuccess {
		results, err = a.db.Search(query)
		if err != nil {
			return nil, err
		}
	}

	// 3. Post-process: Convert to AISearchResult and run Go Smart Reranker
	var aiResults []llm.AISearchResult
	for _, r := range results {
		aiResults = append(aiResults, llm.AISearchResult{
			Path:     r.Path,
			Filename: r.Filename,
			Summary:  r.Summary,
		})
	}

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

// DownloadLLM extracts the embedded GGUF model and the appropriate llama-cli runtimes locally
func (a *App) DownloadLLM() (string, error) {
	modelsDir := filepath.Join(a.appDataDir, "models")
	if err := os.MkdirAll(modelsDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create models directory: %w", err)
	}

	modelPath := filepath.Join(modelsDir, llm.ModelName)

	// 668788096 bytes is the exact size of the model
	const modelSize = int64(668788096)

	// Check if already extracted and complete
	if info, err := os.Stat(modelPath); err == nil && info.Size() == modelSize {
		// Run platform-specific runtime extraction check
		platform := runtime.Environment(a.ctx).Platform
		binDir := filepath.Join(a.appDataDir, "bin")
		cliName := "llama-cli"
		if platform == "windows" {
			cliName = "llama-cli.exe"
		}
		cliPath := filepath.Join(binDir, cliName)
		
		// If cli binary doesn't exist, extract it!
		if _, errCli := os.Stat(cliPath); errCli != nil {
			srcDir := ""
			if platform == "darwin" {
				srcDir = "model/bin/darwin-arm64"
			} else if platform == "windows" {
				srcDir = "model/bin/windows-amd64"
			}

			if srcDir != "" {
				runtime.EventsEmit(a.ctx, "llm-download-start", "Extracting local AI engine runtimes...")
				err = a.extractEmbedDir(srcDir, binDir)
				if err != nil {
					runtime.EventsEmit(a.ctx, "llm-download-error", fmt.Sprintf("failed to extract AI engine: %v", err))
					return "", err
				}
			}
		}

		runtime.EventsEmit(a.ctx, "llm-download-complete", modelPath)
		return modelPath, nil
	}

	runtime.EventsEmit(a.ctx, "llm-download-start", "Extracting local model...")

	// Open the embedded file
	embedFile, err := a.modelFS.Open("model/" + llm.ModelName)
	if err != nil {
		runtime.EventsEmit(a.ctx, "llm-download-error", fmt.Sprintf("failed to open embedded model: %v", err))
		return "", err
	}
	defer embedFile.Close()

	// Create temporary target file
	tmpPath := modelPath + ".tmp"
	out, err := os.Create(tmpPath)
	if err != nil {
		runtime.EventsEmit(a.ctx, "llm-download-error", fmt.Sprintf("failed to create temp file: %v", err))
		return "", err
	}
	defer out.Close()

	// Extract in chunks and emit progress
	buf := make([]byte, 1024*1024) // 1MB buffer
	var totalWritten int64

	for {
		n, readErr := embedFile.Read(buf)
		if n > 0 {
			_, writeErr := out.Write(buf[:n])
			if writeErr != nil {
				runtime.EventsEmit(a.ctx, "llm-download-error", fmt.Sprintf("failed to write temp file: %v", writeErr))
				return "", writeErr
			}
			totalWritten += int64(n)
			
			percent := int((float64(totalWritten) / float64(modelSize)) * 100)
			runtime.EventsEmit(a.ctx, "llm-download-progress", map[string]interface{}{
				"Total":      uint64(modelSize),
				"Downloaded": uint64(totalWritten),
				"Percentage": percent,
			})
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			runtime.EventsEmit(a.ctx, "llm-download-error", fmt.Sprintf("failed to read embedded model: %v", readErr))
			return "", readErr
		}
	}

	out.Close() // Close before renaming

	if err := os.Rename(tmpPath, modelPath); err != nil {
		runtime.EventsEmit(a.ctx, "llm-download-error", fmt.Sprintf("failed to finalize model: %v", err))
		return "", err
	}

	// Extract platform-specific llama-cli binaries
	platform := runtime.Environment(a.ctx).Platform
	binDir := filepath.Join(a.appDataDir, "bin")

	srcDir := ""
	if platform == "darwin" {
		srcDir = "model/bin/darwin-arm64"
	} else if platform == "windows" {
		srcDir = "model/bin/windows-amd64"
	}

	if srcDir != "" {
		runtime.EventsEmit(a.ctx, "llm-download-start", "Extracting local AI engine runtimes...")
		err = a.extractEmbedDir(srcDir, binDir)
		if err != nil {
			runtime.EventsEmit(a.ctx, "llm-download-error", fmt.Sprintf("failed to extract AI engine: %v", err))
			return "", err
		}
	}

	runtime.EventsEmit(a.ctx, "llm-download-complete", modelPath)
	return modelPath, nil
}

// extractEmbedDir extracts a directory from the embedded filesystem recursively
func (a *App) extractEmbedDir(srcDir, destDir string) error {
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return err
	}

	return fs.WalkDir(a.modelFS, srcDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Get relative path from srcDir
		relPath, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}

		if relPath == "." {
			return nil
		}

		targetPath := filepath.Join(destDir, relPath)

		if d.IsDir() {
			return os.MkdirAll(targetPath, 0755)
		}

		// Read embedded file
		data, err := a.modelFS.ReadFile(path)
		if err != nil {
			return err
		}

		// Write target file with 0755 permissions for binaries
		err = os.WriteFile(targetPath, data, 0755)
		if err != nil {
			return err
		}

		// On macOS, strip Gatekeeper quarantine flags so they run silently
		if runtime.Environment(a.ctx).Platform == "darwin" {
			exec.Command("xattr", "-d", "com.apple.quarantine", targetPath).Run()
		}

		return nil
	})
}

// GetSettings gets settings (mock for now)
func (a *App) GetSettings() map[string]interface{} {
	return map[string]interface{}{
		"model_path": filepath.Join(a.appDataDir, "models", llm.ModelName),
		"platform":   runtime.Environment(a.ctx).Platform,
	}
}

