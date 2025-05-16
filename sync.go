package mongospecgpt

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/prestonvasquez/mongo-spec-gpt/internal/mongoutil"
	"github.com/sirupsen/logrus"
	"github.com/tmc/langchaingo/embeddings"
	"github.com/tmc/langchaingo/llms/openai"
	"github.com/tmc/langchaingo/schema"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

const (
	gitHubAPIBase = "https://api.github.com/repos"
	repoOwner     = "mongodb"
	repoName      = "specifications"
	chunkSize     = 800
	chunkOverlap  = 100
)

type GitHubFile struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	Type        string `json:"type"`
	DownloadURL string `json:"download_url"`
}

// Defines the document structure
type Document struct {
	PageContent string            `bson:"text"`
	Embedding   []float32         `bson:"embedding"`
	Metadata    map[string]string `bson:"metadata"`
}

type SyncOptions struct {
	Chunker Chunker
}

type SyncOption func(*SyncOptions)

func WithChunker(chunker Chunker) SyncOption {
	return func(opts *SyncOptions) {
		opts.Chunker = chunker
	}
}

func Sync(ctx context.Context, opts ...SyncOption) error {
	syncOpts := SyncOptions{
		Chunker: NewMarkdownChunker(mongoutil.DefaultOpenAIEmbeddingModel, chunkSize, chunkOverlap),
		//Chunker: NewSentenceChunker(chunkSize, chunkOverlap),
	}
	for _, opt := range opts {
		if opt != nil {
			opt(&syncOpts)
		}
	}

	if syncOpts.Chunker == nil {
		return fmt.Errorf("chunker is not set")
	}

	logrus.Info("Syncing .md files from GitHub repository...")

	files, err := getFiles(repoOwner, repoName, "")
	if err != nil {
		return fmt.Errorf("\nFailed to fetch .md files: %w", err)
	}

	chunks, err := chunkFiles(syncOpts.Chunker, files)
	if err != nil {
		return fmt.Errorf("\nFailed to chunk files: %w", err)
	}
	err = insertFiles(ctx, chunks)

	if err != nil {
		return fmt.Errorf("\nFailed to insert files: %w", err)
	}

	logrus.Info("Successfully synced .md files to MongoDB.")

	return nil
}

// Chunk files for document insertion
func chunkFiles(chunker Chunker, files map[string]string) ([]schema.Document, error) {
	logrus.Infof("Chunking %d files...", len(files))
	values := make([]string, 0, len(files))
	metadata := make([]map[string]any, 0, len(files))

	for path, content := range files {
		values = append(values, content)
		meta := map[string]any{
			"source": strings.Split(path, "/")[len(strings.Split(path, "/"))-1],
		}
		metadata = append(metadata, meta)
		logrus.Infof("Chunking file: %s", path)
	}

	docs, err := chunker.Chunk(values, metadata)
	if err != nil {
		return nil, err
	}

	logrus.Infof("Chunked %d files into %d documents.", len(files), len(docs))

	return docs, nil
}

// Embed and insert chunks as documents
func insertFiles(ctx context.Context, docs []schema.Document) error {
	logrus.Info("Inserting documents into MongoDB...")
	client, _ := mongo.Connect(options.Client().ApplyURI(os.Getenv("MONGODB_URI")))

	defer func() {
		if err := client.Disconnect(ctx); err != nil {
			fmt.Errorf("\nFailed disconnecting the client: %w", err)
		}
	}()

	coll := client.Database(mongoutil.DefaultDatabaseName).Collection(mongoutil.DefaultNamespace)

	llm, err := openai.New(
		openai.WithModel(mongoutil.DefaultOpenAIEmbeddingModel),
		openai.WithEmbeddingModel(mongoutil.DefaultOpenAIEmbeddingModel),
		openai.WithAPIType(openai.APITypeAzure),
	)

	if err != nil {
		return fmt.Errorf("\nFailed to create an embedder client: %w", err)
	}

	embedder, err := embeddings.NewEmbedder(llm)
	if err != nil {
		return fmt.Errorf("\nFailed to create an embedder: %w", err)
	}

	store, err := mongoutil.Store(ctx, client, embedder)
	if err != nil {
		return fmt.Errorf("\nFailed to create a store: %w", err)
	}

	coll.DeleteMany(ctx, bson.D{})

	_, err = store.AddDocuments(ctx, docs)

	if err != nil {
		return fmt.Errorf("\nFailed adding documents: %w", err)
	}

	logrus.Infof("Inserted %d documents into MongoDB.", len(docs))

	return nil
}

// Recursively fetch all .md files from a GitHub repository.
func getFiles(owner, repo, dir string) (map[string]string, error) {
	url := fmt.Sprintf("%s/%s/%s/contents/%s", gitHubAPIBase, owner, repo, dir)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", fmt.Sprintf("token %s", os.Getenv("GITHUB_PAT")))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("\nFailed to fetch contents: %v", resp.Status)
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
		// Only fetch .md files that are within the source/ subdirectory
		if file.Type == "file" && strings.HasSuffix(file.Name, ".md") && strings.Contains(file.Path, "source/") && !strings.Contains(file.Path, "test") {
			content, err := fetchFileContent(file.DownloadURL)
			if err != nil {
				return nil, err
			}
			mdFiles[file.Path] = content
			logrus.Infof("Fetched file: %s", file.Path)
		} else if file.Type == "dir" {
			// Recurse for directories
			subDirFiles, err := getFiles(owner, repo, file.Path)
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

// Fetch the contents of a file from its URL.
func fetchFileContent(downloadURL string) (string, error) {
	resp, err := http.Get(downloadURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("\nFailed to fetch file content: %v", resp.Status)
	}

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(content), nil
}
