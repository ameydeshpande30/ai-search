package db

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite"
)

type DB struct {
	conn *sql.DB
}

type SearchResult struct {
	Path     string   `json:"path"`
	Filename string   `json:"filename"`
	Tags     []string `json:"tags"`
	Summary  string   `json:"summary"`
}

func InitDB(appDataDir string) (*DB, error) {
	dbPath := filepath.Join(appDataDir, "index.db")
	conn, err := sql.Open("sqlite", dbPath+"?mode=rwc&_pragma=journal_mode(WAL)")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Create tables with FTS5
	// documents contains the original file paths and metadata
	// fts_index is the virtual table for fast full-text search
	query := `
	CREATE TABLE IF NOT EXISTS documents (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		path TEXT UNIQUE NOT NULL,
		filename TEXT NOT NULL,
		summary TEXT,
		tags TEXT,
		last_modified INTEGER
	);

	CREATE VIRTUAL TABLE IF NOT EXISTS fts_index USING fts5(
		filename,
		summary,
		tags,
		content='documents',
		content_rowid='id'
	);

	-- Triggers to automatically update the FTS table
	CREATE TRIGGER IF NOT EXISTS documents_ai AFTER INSERT ON documents BEGIN
		INSERT INTO fts_index(rowid, filename, summary, tags) 
		VALUES (new.id, new.filename, new.summary, new.tags);
	END;

	CREATE TRIGGER IF NOT EXISTS documents_au AFTER UPDATE ON documents BEGIN
		INSERT INTO fts_index(fts_index, rowid, filename, summary, tags) 
		VALUES ('delete', old.id, old.filename, old.summary, old.tags);
		INSERT INTO fts_index(rowid, filename, summary, tags) 
		VALUES (new.id, new.filename, new.summary, new.tags);
	END;

	CREATE TRIGGER IF NOT EXISTS documents_ad AFTER DELETE ON documents BEGIN
		INSERT INTO fts_index(fts_index, rowid, filename, summary, tags) 
		VALUES ('delete', old.id, old.filename, old.summary, old.tags);
	END;
	`

	if _, err := conn.Exec(query); err != nil {
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return &DB{conn: conn}, nil
}

// Search performs a fast FTS5 keyword search
func (db *DB) Search(query string) ([]SearchResult, error) {
	if query == "" {
		return nil, nil
	}

	// Prepare FTS5 query: split by space and append * to each word for prefix matching
	words := strings.Fields(query)
	var ftsWords []string
	for _, w := range words {
		// Remove any quotes from user input to prevent syntax errors
		w = strings.ReplaceAll(w, `"`, "")
		if w != "" {
			ftsWords = append(ftsWords, `"`+w+`"*`)
		}
	}
	ftsQuery := strings.Join(ftsWords, " AND ")

	if ftsQuery == "" {
		return nil, nil
	}

	sqlQuery := `
		SELECT d.path, d.filename, d.summary, d.tags
		FROM fts_index f
		JOIN documents d ON d.id = f.rowid
		WHERE fts_index MATCH ?
		ORDER BY rank
		LIMIT 20
	`

	rows, err := db.conn.Query(sqlQuery, ftsQuery)
	if err != nil {
		return nil, fmt.Errorf("search query failed: %w", err)
	}
	defer rows.Close()

	var results []SearchResult
	for rows.Next() {
		var res SearchResult
		var tagsStr string
		if err := rows.Scan(&res.Path, &res.Filename, &res.Summary, &tagsStr); err != nil {
			continue
		}
		res.Tags = parseTags(tagsStr)
		results = append(results, res)
	}

	// Fallback to fuzzy LIKE search if FTS returns no results
	if len(results) == 0 && len(words) > 0 {
		fuzzyQuery := "%" + strings.Join(words, "%") + "%"
		sqlLikeQuery := `
			SELECT path, filename, summary, tags
			FROM documents
			WHERE filename LIKE ? OR summary LIKE ? OR tags LIKE ?
			LIMIT 20
		`
		rowsLike, err := db.conn.Query(sqlLikeQuery, fuzzyQuery, fuzzyQuery, fuzzyQuery)
		if err == nil {
			defer rowsLike.Close()
			for rowsLike.Next() {
				var res SearchResult
				var tagsStr string
				if err := rowsLike.Scan(&res.Path, &res.Filename, &res.Summary, &tagsStr); err == nil {
					res.Tags = parseTags(tagsStr)
					results = append(results, res)
				}
			}
		}
	}

	return results, nil
}

// AddDocument upserts a document into the index
func (db *DB) AddDocument(path, filename, summary string, tags []string, lastModified int64) error {
	tagsStr := joinTags(tags)
	
	query := `
		INSERT INTO documents (path, filename, summary, tags, last_modified)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(path) DO UPDATE SET
			filename = excluded.filename,
			summary = excluded.summary,
			tags = excluded.tags,
			last_modified = excluded.last_modified
	`

	_, err := db.conn.Exec(query, path, filename, summary, tagsStr, lastModified)
	return err
}

func parseTags(tagsStr string) []string {
	if tagsStr == "" {
		return nil
	}
	return strings.Split(tagsStr, ",")
}

func joinTags(tags []string) string {
	return strings.Join(tags, ",")
}

// SeedMockData adds some mock data to the database for testing
func (db *DB) SeedMockData() error {
	mockDocs := []struct {
		Path     string
		Filename string
		Summary  string
		Tags     []string
	}{
		{"/Users/ameyd/Documents/Finance/2024/Q1_Invoice_Adobe.pdf", "Q1_Invoice_Adobe.pdf", "Adobe subscription invoice for Jan-Mar 2024", []string{"Finance", "Invoice", "Adobe", "2024"}},
		{"/Users/ameyd/Documents/Legal/Lease_Agreement_2023.docx", "Lease_Agreement_2023.docx", "Residential lease agreement for downtown apartment", []string{"Legal", "Contract", "Housing"}},
		{"/Users/ameyd/Documents/Taxes/2023/tax_returns_summary.txt", "tax_returns_summary.txt", "Summary of 2023 tax returns and deductions", []string{"Taxes", "Finance", "2023"}},
		{"/Users/ameyd/Documents/Codes/ai-search-app/main.go", "main.go", "Main entry point for local AI search application", []string{"Code", "Go", "Project"}},
		{"/Users/ameyd/Downloads/Resume_v4.pdf", "Resume_v4.pdf", "Updated software engineering resume with Wails experience", []string{"Career", "Resume"}},
	}

	for _, doc := range mockDocs {
		err := db.AddDocument(doc.Path, doc.Filename, doc.Summary, doc.Tags, 0)
		if err != nil {
			return err
		}
	}
	return nil
}
