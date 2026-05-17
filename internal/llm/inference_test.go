package llm

import (
	"testing"
)

func TestSmartRerank_FiltersNodeModules(t *testing.T) {
	results := []AISearchResult{
		{Path: "/Users/test/Documents/codes/myapp/node_modules/@babel/generator/typescript.js", Filename: "typescript.js", Summary: "exports.TSAnyKeyword = TSAnyKeyword"},
		{Path: "/Users/test/Documents/myproject/main.ts", Filename: "main.ts", Summary: "import typescript compiler for the project"},
		{Path: "/Users/test/Documents/codes/myapp/src/App.tsx", Filename: "App.tsx", Summary: "main typescript react application file"},
	}

	ranked := SmartRerank("typescript", results)

	// node_modules result should be filtered out
	nodeModulesPath := "/Users/test/Documents/codes/myapp/node_modules/@babel/generator/typescript.js"
	for _, r := range ranked {
		if r.Path == nodeModulesPath {
			t.Errorf("node_modules file should have been filtered, but got: %s", r.Path)
		}
	}

	// User files should remain
	if len(ranked) == 0 {
		t.Fatal("Expected at least 1 result after filtering")
	}
	if ranked[0].Filename != "main.ts" && ranked[0].Filename != "App.tsx" {
		t.Errorf("Expected user file to be top result, got: %s", ranked[0].Filename)
	}
}

func TestSmartRerank_FiltersJDKAndSDKPaths(t *testing.T) {
	results := []AISearchResult{
		{Path: "/Users/test/Documents/apps/JAVA_JDK/zulu-16.jdk/Contents/Home/legal/java.base/cldr.md", Filename: "cldr.md", Summary: "UNICODE LICENSE AGREEMENT syntax tax report"},
		{Path: "/Users/test/Documents/Finance/tax_report_2024.pdf", Filename: "tax_report_2024.pdf", Summary: "Annual tax report for fiscal year 2024"},
		{Path: "/Users/test/Library/Developer/Xcode/DerivedData/myapp/Build/main.swift", Filename: "main.swift", Summary: "tax calculation function"},
	}

	ranked := SmartRerank("tax report", results)

	if len(ranked) == 0 {
		t.Fatal("Expected at least 1 result")
	}

	// The PDF tax report should be #1
	if ranked[0].Filename != "tax_report_2024.pdf" {
		t.Errorf("Expected tax_report_2024.pdf as top result, got: %s (score: %.1f)", ranked[0].Filename, ranked[0].Score)
	}

	// The JDK cldr.md should be heavily penalized
	for _, r := range ranked {
		if r.Filename == "cldr.md" && r.Score > 0 {
			t.Errorf("JDK license file should have negative score, got: %.1f for %s", r.Score, r.Path)
		}
	}
}

func TestSmartRerank_WordBoundaryMatching(t *testing.T) {
	results := []AISearchResult{
		{Path: "/Users/test/docs/license.md", Filename: "license.md", Summary: "This is a syntax reference with no actual tax information"},
		{Path: "/Users/test/docs/taxes.txt", Filename: "taxes.txt", Summary: "My tax returns and tax deductions for 2024"},
	}

	ranked := SmartRerank("tax", results)

	if len(ranked) < 2 {
		t.Fatal("Expected 2 results")
	}

	// taxes.txt should rank higher because it has actual "tax" as a word, not embedded in "syntax"
	if ranked[0].Filename != "taxes.txt" {
		t.Errorf("Expected taxes.txt as top result, got: %s (score: %.1f)", ranked[0].Filename, ranked[0].Score)
	}
}

func TestSmartRerank_PrefersUserDocuments(t *testing.T) {
	results := []AISearchResult{
		{Path: "/Users/test/codes/project/config.json", Filename: "config.json", Summary: "amazon aws configuration settings"},
		{Path: "/Users/test/Documents/amazon_invoice.pdf", Filename: "amazon_invoice.pdf", Summary: "Amazon order invoice for electronics"},
		{Path: "/Users/test/codes/project/src/api.ts", Filename: "api.ts", Summary: "connects to amazon s3 bucket"},
	}

	ranked := SmartRerank("amazon", results)

	if len(ranked) == 0 {
		t.Fatal("Expected results")
	}

	// The PDF invoice should rank highest (filename match + high-value extension + content match)
	if ranked[0].Filename != "amazon_invoice.pdf" {
		t.Errorf("Expected amazon_invoice.pdf as top result, got: %s (score: %.1f)", ranked[0].Filename, ranked[0].Score)
	}
}

func TestSmartRerank_EmptyInput(t *testing.T) {
	ranked := SmartRerank("test", nil)
	if ranked != nil {
		t.Errorf("Expected nil for empty input, got: %v", ranked)
	}

	ranked = SmartRerank("test", []AISearchResult{})
	if ranked != nil {
		t.Errorf("Expected nil for empty slice, got: %v", ranked)
	}
}

func TestBuildPrompt(t *testing.T) {
	results := []AISearchResult{
		{Path: "/Users/test/docs/resume.pdf", Filename: "resume.pdf", Summary: "Software engineer resume"},
	}

	prompt := BuildPrompt("resume", results)

	if prompt == "" {
		t.Fatal("Prompt should not be empty")
	}

	// Verify prompt contains the query and result
	if !containsStr(prompt, "resume") {
		t.Error("Prompt should contain the query")
	}
	if !containsStr(prompt, "resume.pdf") {
		t.Error("Prompt should contain the filename")
	}
}

func containsStr(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && indexOf(s, substr) >= 0
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

func TestBuildSQLPrompt(t *testing.T) {
	prompt := BuildSQLPrompt("pdf from last week", 1715978400)
	if prompt == "" {
		t.Fatal("SQL Prompt should not be empty")
	}

	if !containsStr(prompt, "pdf from last week") {
		t.Error("SQL Prompt should contain user query")
	}

	if !containsStr(prompt, "Table: documents") {
		t.Error("SQL Prompt should explain table schema")
	}

	if !containsStr(prompt, "1715978400") {
		t.Error("SQL Prompt should inject current timestamp")
	}
}

func TestSanitizeSQLQuery(t *testing.T) {
	// Valid SELECT queries
	validQueries := []string{
		"SELECT * FROM documents WHERE filename LIKE '%.pdf';",
		"```sql\nSELECT * FROM documents WHERE last_modified >= 1700000000;\n```",
		"   select * from documents   ",
	}

	for _, q := range validQueries {
		sanitized, err := SanitizeSQLQuery(q)
		if err != nil {
			t.Errorf("Expected valid query to pass, got error: %v for %s", err, q)
		}
		if !containsStr(sanitized, "select") && !containsStr(sanitized, "SELECT") {
			t.Errorf("Sanitized query should retain SELECT: %s", sanitized)
		}
	}

	// Unsafe or invalid queries
	invalidQueries := []string{
		"DROP TABLE documents;",
		"SELECT * FROM documents; DELETE FROM documents;",
		"INSERT INTO documents VALUES (1, 'a.txt', 'b.txt', 'c', 'd', 0);",
		"UPDATE documents SET tags = 'hack' WHERE id = 1;",
	}

	for _, q := range invalidQueries {
		_, err := SanitizeSQLQuery(q)
		if err == nil {
			t.Errorf("Expected query to fail sanitization, but it passed: %s", q)
		}
	}
}
