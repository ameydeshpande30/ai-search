package llm

import (
	"fmt"
	"math"
	"path/filepath"
	"sort"
	"strings"
)

// AISearchResult represents a reranked result from the AI engine
type AISearchResult struct {
	Path      string  `json:"path"`
	Filename  string  `json:"filename"`
	Summary   string  `json:"summary"`
	AIReason  string  `json:"aiReason"`
	Score     float64 `json:"score"`
}

// junkDirPatterns are universal directory names that always indicate non-user content
var junkDirPatterns = []string{
	"node_modules", "__pycache__", ".cache",
}

// junkFilePatterns are universal files that are never user-relevant
var junkFilePatterns = []string{
	"package-lock.json", "yarn.lock", "go.sum",
	".DS_Store", "Thumbs.db",
}

// highValueExtensions are file types commonly created by humans (not tools)
var highValueExtensions = map[string]float64{
	".pdf": 2.0, ".docx": 2.0, ".doc": 2.0, ".xlsx": 2.0, ".xls": 2.0,
	".pptx": 2.0, ".ppt": 2.0, ".txt": 1.5, ".md": 1.2, ".csv": 1.5,
	".jpg": 1.2, ".png": 1.2, ".mp4": 1.2, ".mp3": 1.2,
}

// BuildPrompt constructs the AI reranking prompt
func BuildPrompt(query string, results []AISearchResult) string {
	var sb strings.Builder
	sb.WriteString("<|system|>\n")
	sb.WriteString("You are a highly intelligent local file search assistant.\n")
	sb.WriteString("Your task is to review the following search results and select ONLY the most relevant files that match the user's query.\n")
	sb.WriteString("RULES:\n")
	sb.WriteString("- Filter out any dependency files (node_modules, dist, build artifacts, package managers)\n")
	sb.WriteString("- Filter out any compiled or minified code\n")
	sb.WriteString("- Prefer user-created documents (PDFs, text files, spreadsheets, personal code)\n")
	sb.WriteString("- Return ONLY the indices of the top 3 most relevant results as comma-separated numbers\n")
	sb.WriteString("- If no results are relevant, return 'NONE'\n")
	sb.WriteString("</s>\n")
	sb.WriteString("<|user|>\n")
	sb.WriteString(fmt.Sprintf("Query: \"%s\"\n\nResults:\n", query))

	for i, r := range results {
		summary := r.Summary
		if len(summary) > 200 {
			summary = summary[:200] + "..."
		}
		sb.WriteString(fmt.Sprintf("%d. [%s] %s\n   Content: %s\n\n", i+1, r.Filename, r.Path, summary))
	}

	sb.WriteString("</s>\n<|assistant|>\n")
	return sb.String()
}

// SmartRerank uses heuristic scoring to intelligently rerank search results
func SmartRerank(query string, results []AISearchResult) []AISearchResult {
	if len(results) == 0 {
		return nil
	}

	queryLower := strings.ToLower(query)
	queryWords := strings.Fields(queryLower)

	for i := range results {
		score := 0.0
		pathLower := strings.ToLower(results[i].Path)
		filenameLower := strings.ToLower(results[i].Filename)
		summaryLower := strings.ToLower(results[i].Summary)

		// Normalize to forward slashes for cross-platform consistency
		pathNormalized := filepath.ToSlash(pathLower)
		pathParts := strings.Split(pathNormalized, "/")

		// 1. PENALTY: Known junk directory in path
		isJunk := false
		for _, junkDir := range junkDirPatterns {
			if strings.Contains(pathNormalized, "/"+junkDir+"/") {
				score -= 100
				isJunk = true
				break
			}
		}

		// 2. PENALTY: Known junk files
		for _, junkFile := range junkFilePatterns {
			if filenameLower == strings.ToLower(junkFile) {
				score -= 50
				isJunk = true
				break
			}
		}

		// 3. PENALTY: Generalized "likely not user content" — deep paths with
		//    build/dist/vendor/legal/license-type segments
		if !isJunk {
			for _, part := range pathParts {
				switch part {
				case "dist", "build", "vendor", "target", "obj",
					"coverage", "legal", "license", "licenses":
					if len(pathParts) > 5 {
						score -= 30
						isJunk = true
					}
				}
			}
		}

		// 4. BOOST: High-value file extension
		ext := strings.ToLower(filepath.Ext(results[i].Filename))
		if boost, ok := highValueExtensions[ext]; ok {
			score += boost * 10
		}

		// 5. BOOST: Query word appears in filename (strongest positive signal)
		for _, word := range queryWords {
			if strings.Contains(filenameLower, word) {
				score += 30
			}
		}

		// 6. BOOST: Query word appears in the path (user's own directory structure)
		for _, word := range queryWords {
			// Only count path segments, not the filename again
			dirPath := strings.ToLower(filepath.ToSlash(filepath.Dir(results[i].Path)))
			if strings.Contains(dirPath, word) {
				score += 10
			}
		}

		// 7. BOOST: Query word appears in document content (WORD BOUNDARY matching)
		for _, word := range queryWords {
			count := countWholeWord(summaryLower, word)
			if count > 0 {
				score += math.Min(float64(count)*3, 25)
			}
		}

		// 8. PENALTY: Very deep nesting (likely dependency/system file)
		depth := strings.Count(filepath.ToSlash(results[i].Path), "/")
		if depth > 6 {
			score -= float64(depth-6) * 3
		}

		// 9. Generate AI reason
		if !isJunk {
			reason := generateReason(query, results[i], queryWords)
			results[i].AIReason = reason
		} else {
			results[i].AIReason = "Filtered: dependency/system file"
		}

		results[i].Score = score
	}

	// Sort by score descending
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	// Filter out junk and return top 5
	var filtered []AISearchResult
	for _, r := range results {
		if r.Score > -10 {
			filtered = append(filtered, r)
		}
		if len(filtered) >= 5 {
			break
		}
	}

	return filtered
}

// countWholeWord counts occurrences of a word with word boundaries
// "tax" matches "tax" and "taxes" but NOT "syntax"
func countWholeWord(text, word string) int {
	count := 0
	searchFrom := 0
	for {
		idx := strings.Index(text[searchFrom:], word)
		if idx == -1 {
			break
		}
		absIdx := searchFrom + idx

		// Check left boundary: must be start of string or non-alphanumeric
		leftOk := absIdx == 0 || !isAlphaNum(text[absIdx-1])
		// Right boundary: word end, or the next char continues the word (like "taxes" from "tax") — that's OK
		// But "syntax" contains "tax" at position 3, where left char is 'n' which is alphanumeric — NOT OK

		if leftOk {
			count++
		}
		searchFrom = absIdx + len(word)
		if searchFrom >= len(text) {
			break
		}
	}
	return count
}

func isAlphaNum(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9')
}

// generateReason creates a human-readable explanation of why this result matched
func generateReason(query string, result AISearchResult, queryWords []string) string {
	reasons := []string{}

	filenameLower := strings.ToLower(result.Filename)
	summaryLower := strings.ToLower(result.Summary)

	for _, word := range queryWords {
		if strings.Contains(filenameLower, word) {
			reasons = append(reasons, fmt.Sprintf("Filename contains \"%s\"", word))
		}
		count := countWholeWord(summaryLower, word)
		if count > 0 {
			reasons = append(reasons, fmt.Sprintf("Found \"%s\" %d time(s) in content", word, count))
		}
	}

	ext := strings.ToLower(filepath.Ext(result.Filename))
	if _, ok := highValueExtensions[ext]; ok {
		reasons = append(reasons, fmt.Sprintf("High-value document type (%s)", ext))
	}

	if len(reasons) == 0 {
		return "Partial match found in file content"
	}

	return strings.Join(reasons, " · ")
}

// BuildSQLPrompt builds the SQLite text-to-SQL GGUF prompt.
func BuildSQLPrompt(query string, now int64) string {
	var sb strings.Builder
	sb.WriteString("<|system|>\n")
	sb.WriteString("You are a highly precise SQLite assistant.\n")
	sb.WriteString("Translate the user's natural language file search query into a valid, single SQLite SELECT query.\n\n")
	sb.WriteString("DATABASE SCHEMA:\n")
	sb.WriteString("Table: documents\n")
	sb.WriteString("Columns:\n")
	sb.WriteString("- path (TEXT)\n")
	sb.WriteString("- filename (TEXT)\n")
	sb.WriteString("- summary (TEXT)\n")
	sb.WriteString("- tags (TEXT)\n")
	sb.WriteString("- last_modified (INTEGER, unix timestamp in seconds)\n\n")
	sb.WriteString(fmt.Sprintf("CURRENT UNIX TIMESTAMP: %d (May 18, 2026)\n\n", now))
	sb.WriteString("RULES:\n")
	sb.WriteString("- Output ONLY the single SQLite SELECT statement starting with 'SELECT * FROM documents'.\n")
	sb.WriteString("- Never include any explanations, conversation, or markdown code blocks (like ```sql).\n")
	sb.WriteString("- Use LIKE wildcards for file extensions or keyword matching (e.g. filename LIKE '%.pdf').\n")
	sb.WriteString("- For date/time arithmetic, compare 'last_modified' with appropriate math from the CURRENT UNIX TIMESTAMP.\n")
	sb.WriteString("</s>\n<|user|>\n")
	sb.WriteString(fmt.Sprintf("Query: \"%s\"\n", query))
	sb.WriteString("</s>\n<|assistant|>\n")
	return sb.String()
}

// SanitizeSQLQuery ensures the generated SQL query is a valid read-only SELECT statement.
func SanitizeSQLQuery(sqlStr string) (string, error) {
	// Clean markdown block wrappers if LLM returned them
	sqlStr = strings.TrimSpace(sqlStr)
	sqlStr = strings.TrimPrefix(sqlStr, "```sql")
	sqlStr = strings.TrimPrefix(sqlStr, "```")
	sqlStr = strings.TrimSuffix(sqlStr, "```")
	sqlStr = strings.TrimSpace(sqlStr)

	// Strip trailing semicolon if present
	sqlStr = strings.TrimSuffix(sqlStr, ";")

	sqlLower := strings.ToLower(sqlStr)

	// Security validation: Must ONLY be a SELECT query
	if !strings.HasPrefix(sqlLower, "select") {
		return "", fmt.Errorf("unsafe or invalid query: must start with SELECT")
	}

	// Prevent SQL injection / modification queries
	dangerKeywords := []string{
		"insert", "update", "delete", "drop", "alter", "create", "replace",
		"truncate", "grant", "revoke", "exec", "union", "attach", "detach",
	}

	for _, keyword := range dangerKeywords {
		if strings.Contains(sqlLower, " "+keyword+" ") || strings.Contains(sqlLower, ";"+keyword) || strings.HasPrefix(sqlLower, keyword) {
			return "", fmt.Errorf("security violation: query contains dangerous keyword '%s'", keyword)
		}
	}

	return sqlStr + ";", nil
}
