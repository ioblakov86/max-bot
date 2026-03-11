package joomla

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// JoomlaClient provides integration with Joomla CMS for updating articles
type JoomlaClient struct {
	ArticleIDs  []int64
	SiteURL     string
	AdminURL    string
	Username    string
	Password    string
	APIToken    string
	PythonPath  string
	ScriptPath  string
	MaxRetries  int
	RetryDelay  time.Duration
}

// Employee represents an employee from AI analysis
type Employee struct {
	Position string `json:"position"`
	FullName string `json:"full_name"`
}

// Dates represents absence dates
type Dates struct {
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`
}

// AnalysisResult represents the AI analysis result
type AnalysisResult struct {
	OriginalMessage string   `json:"original_message"`
	IsValid         bool     `json:"is_valid"`
	Employee        Employee `json:"employee"`
	AbsenceType     string   `json:"absence_type"`
	Dates           Dates    `json:"dates"`
	Status          string   `json:"status"`
	Substitute      string   `json:"substitute"`
}

// Change represents a planned change to an article
type Change struct {
	ArticleID int64  `json:"article_id"`
	Doctor    string `json:"doctor"`
	Action    string `json:"action"` // "add" or "remove"
	OldHTML   string `json:"old_html"`
	NewHTML   string `json:"new_html"`
}

// AnalyzeResponse represents the response from joomla_analyzer.py analyze command
type AnalyzeResponse struct {
	Success   bool     `json:"success"`
	Employee  Employee `json:"employee"`
	Changes   []Change `json:"changes"`
	Errors    []string `json:"errors"`
	Message   string   `json:"message,omitempty"`
}

// ApplyResponse represents the response from joomla_analyzer.py apply command
type ApplyResponse struct {
	Success         bool    `json:"success"`
	UpdatedArticles []int64 `json:"updated_articles"`
	Errors          []string `json:"errors"`
}

// NewJoomlaClient creates a new Joomla client from environment variables
func NewJoomlaClient() *JoomlaClient {
	// Parse article IDs from comma-separated string
	articleIDsStr := os.Getenv("JOOMLA_ARTICLE_IDS")
	articleIDs := make([]int64, 0)
	if articleIDsStr != "" {
		for _, idStr := range strings.Split(articleIDsStr, ",") {
			idStr = strings.TrimSpace(idStr)
			if idStr != "" {
				var id int64
				fmt.Sscanf(idStr, "%d", &id)
				articleIDs = append(articleIDs, id)
			}
		}
	}

	// Default to standard IDs if not specified
	if len(articleIDs) == 0 {
		articleIDs = []int64{1025, 1027, 1028, 1029}
	}

	// Find the script path - try absolute path first (for Docker), then relative
	scriptPath := "/root/joomla/joomla_analyzer.py"
	// Try different paths
	possiblePaths := []string{
		"/root/joomla/joomla_analyzer.py",  // Absolute path (Docker)
		"joomla/joomla_analyzer.py",        // Relative from project root
		"./joomla/joomla_analyzer.py",
		"../joomla/joomla_analyzer.py",
	}
	for _, path := range possiblePaths {
		if _, err := os.Stat(path); err == nil {
			scriptPath = path
			break
		}
	}

	// Find Python executable
	pythonPath := "python3"
	if _, err := exec.LookPath("python3"); err != nil {
		if _, err := exec.LookPath("python"); err == nil {
			pythonPath = "python"
		}
	}

	return &JoomlaClient{
		ArticleIDs:  articleIDs,
		SiteURL:     getEnv("JOOMLA_SITE_URL", "https://plk32.ru"),
		AdminURL:    getEnv("JOOMLA_ADMIN_URL", "https://plk32.ru/administrator"),
		Username:    getEnv("JOOMLA_USERNAME", ""),
		Password:    getEnv("JOOMLA_PASSWORD", ""),
		APIToken:    getEnv("JOOMLA_API_TOKEN", ""),
		PythonPath:  pythonPath,
		ScriptPath:  scriptPath,
		MaxRetries:  3,
		RetryDelay:  2 * time.Second,
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// Analyze analyzes the AI result and returns planned changes
func (c *JoomlaClient) Analyze(analysis AnalysisResult) (*AnalyzeResponse, error) {
	// Skip if status is "Продолжение"
	if analysis.Status == "Продолжение" {
		return &AnalyzeResponse{
			Success: true,
			Message: "Статус \"Продолжение\" - изменения не требуются",
		}, nil
	}

	// Convert analysis to JSON
	analysisJSON, err := json.Marshal(analysis)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal analysis: %w", err)
	}

	// Debug logging
	fmt.Printf("DEBUG: Sending to Python - JSON: %s\n", string(analysisJSON))

	// Run the Python script with retries
	var response AnalyzeResponse
	err = c.runWithRetries(func() error {
		cmd := exec.Command(c.PythonPath, c.ScriptPath, "analyze", "--json", string(analysisJSON))
		cmd.Env = c.buildEnv()
		cmd.Dir = c.GetScriptDirectory()

		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		err := cmd.Run()
		if err != nil {
			fmt.Printf("DEBUG: Python stderr: %s\n", stderr.String())
			fmt.Printf("DEBUG: Python stdout: %s\n", stdout.String())
			return fmt.Errorf("script execution failed: %w, stderr: %s", err, stderr.String())
		}

		// Parse the JSON response
		if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
			return fmt.Errorf("failed to parse response JSON: %w, output: %s", err, stdout.String())
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return &response, nil
}

// Apply applies the planned changes to Joomla
func (c *JoomlaClient) Apply(analysis AnalysisResult, changes []Change) (*ApplyResponse, error) {
	analysisJSON, err := json.Marshal(analysis)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal analysis: %w", err)
	}

	changesJSON, err := json.Marshal(changes)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal changes: %w", err)
	}

	// Run the Python script with retries
	var response ApplyResponse
	err = c.runWithRetries(func() error {
		cmd := exec.Command(c.PythonPath, c.ScriptPath, "apply",
			"--json", string(analysisJSON),
			"--changes", string(changesJSON))
		cmd.Env = c.buildEnv()

		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		err := cmd.Run()
		if err != nil {
			return fmt.Errorf("script execution failed: %w, stderr: %s", err, stderr.String())
		}

		// Parse the JSON response
		if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
			return fmt.Errorf("failed to parse response JSON: %w, output: %s", err, stdout.String())
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return &response, nil
}

// runWithRetries executes a function with retries on failure
func (c *JoomlaClient) runWithRetries(fn func() error) error {
	var lastErr error
	for i := 0; i < c.MaxRetries; i++ {
		lastErr = fn()
		if lastErr == nil {
			return nil
		}
		if i < c.MaxRetries-1 {
			time.Sleep(c.RetryDelay)
		}
	}
	return fmt.Errorf("after %d retries: %w", c.MaxRetries, lastErr)
}

// buildEnv builds the environment variables for the Python script
func (c *JoomlaClient) buildEnv() []string {
	env := os.Environ()
	
	// Add Joomla-specific environment variables
	env = append(env, fmt.Sprintf("JOOMLA_SITE_URL=%s", c.SiteURL))
	env = append(env, fmt.Sprintf("JOOMLA_ADMIN_URL=%s", c.AdminURL))
	env = append(env, fmt.Sprintf("JOOMLA_USERNAME=%s", c.Username))
	env = append(env, fmt.Sprintf("JOOMLA_PASSWORD=%s", c.Password))
	env = append(env, fmt.Sprintf("JOOMLA_API_TOKEN=%s", c.APIToken))
	
	// Convert article IDs to comma-separated string
	ids := make([]string, len(c.ArticleIDs))
	for i, id := range c.ArticleIDs {
		ids[i] = fmt.Sprintf("%d", id)
	}
	env = append(env, fmt.Sprintf("JOOMLA_ARTICLE_IDS=%s", strings.Join(ids, ",")))

	return env
}

// GetScriptPath returns the path to the Python script
func (c *JoomlaClient) GetScriptPath() string {
	return c.ScriptPath
}

// CheckPython checks if Python is available
func CheckPython() (bool, string) {
	// Try python3 first
	if path, err := exec.LookPath("python3"); err == nil {
		return true, path
	}
	// Try python
	if path, err := exec.LookPath("python"); err == nil {
		return true, path
	}
	return false, ""
}

// CheckScript checks if the Python script exists
func (c *JoomlaClient) CheckScript() bool {
	// Try to resolve the script path
	paths := []string{
		c.ScriptPath,
		"joomla/joomla_analyzer.py",
		"./joomla/joomla_analyzer.py",
		"../joomla/joomla_analyzer.py",
	}
	
	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			return true
		}
	}
	
	return false
}

// GetScriptDirectory returns the directory containing the script
func (c *JoomlaClient) GetScriptDirectory() string {
	dir := filepath.Dir(c.ScriptPath)
	// If using absolute path, return it directly
	if filepath.IsAbs(c.ScriptPath) {
		return dir
	}
	// For relative paths, try to get absolute path
	if absDir, err := filepath.Abs(dir); err == nil {
		return absDir
	}
	return dir
}
