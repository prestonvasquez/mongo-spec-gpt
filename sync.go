package mongospecgpt

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

func Sync(ctx context.Context) error {
	owner := "mongodb"
	repo := "specifications"

	mdFiles, err := getMarkdownFiles(owner, repo, "")
	if err != nil {
		fmt.Printf("Failed to fetch .md files: %v\n", err)
		return err
	}

	fmt.Printf("Found %d .md files\n", len(mdFiles))
	for path, _ := range mdFiles {
		fmt.Printf("Path: %s", path)
	}
	return nil
}

const (
	gitHubAPIBase = "https://api.github.com/repos"
)

type GitHubFile struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	Type        string `json:"type"`
	DownloadURL string `json:"download_url"`
}

// getMarkdownFiles recursively fetches all .md files from a GitHub repository.
func getMarkdownFiles(owner, repo, dir string) (map[string]string, error) {
	url := fmt.Sprintf("%s/%s/%s/contents/%s", gitHubAPIBase, owner, repo, dir)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	// Add Authorization token
	req.Header.Set("Authorization", fmt.Sprintf("token %s", os.Getenv("GITHUB_PAT")))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch contents: %v", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var files []GitHubFile
	if err := json.Unmarshal(body, &files); err != nil {
		return nil, err
	}

	mdFiles := make(map[string]string)
	for _, file := range files {
		if file.Type == "file" && strings.HasSuffix(file.Name, ".md") && !strings.Contains(file.Path, "Test") {
			content, err := fetchFileContent(file.DownloadURL)
			if err != nil {
				return nil, err
			}
			mdFiles[file.Path] = content
		} else if file.Type == "dir" {
			// Recurse for directories
			subDirFiles, err := getMarkdownFiles(owner, repo, file.Path)
			if err != nil {
				return nil, err
			}
			for path, content := range subDirFiles {
				mdFiles[path] = content
			}
		}
	}

	return mdFiles, nil
}

// fetchFileContent fetches the content of a file from its download URL.
func fetchFileContent(downloadURL string) (string, error) {
	resp, err := http.Get(downloadURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to fetch file content: %v", resp.Status)
	}

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(content), nil
}
