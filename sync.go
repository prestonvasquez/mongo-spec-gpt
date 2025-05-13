package mongospecgpt

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/tmc/langchaingo/embeddings"
	"github.com/tmc/langchaingo/llms/openai"
	"github.com/tmc/langchaingo/schema"
	"github.com/tmc/langchaingo/textsplitter"
	"github.com/tmc/langchaingo/vectorstores/mongovector"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

const (
	gitHubAPIBase        = "https://api.github.com/repos"
	repoOwner            = "mongodb"
	repoName             = "specifications"
	openAIEmbeddingModel = "text-embedding-3-small"
	openAIEmbeddingDim   = 1536
	similarityAlgorithm  = "dotProduct"
	indexName            = "search_index"
	databaseName         = "spec_gpt"
	collectionName       = "specs"
	chunkSize            = 800
	chunkOverlap         = 100
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

func Sync(ctx context.Context) error {

	files, err := getFiles(repoOwner, repoName, "")
	if err != nil {
		return fmt.Errorf("\nFailed to fetch .md files: %w", err)
	}

	chunks, err := chunkFiles(files)
	if err != nil {
		return fmt.Errorf("\nFailed to chunk files: %w", err)
	}
	err = insertFiles(chunks)

	if err != nil {
		return fmt.Errorf("\nFailed to insert files: %w", err)
	}

	return nil
}

// Chunk files for document insertion
func chunkFiles(files map[string]string) ([]schema.Document, error) {
	values := make([]string, 0, len(files))
	metadata := make([]map[string]any, 0, len(files))

	for k, v := range files {
		values = append(values, v)
		current_metadata := make(map[string]any)
		current_metadata["source"] = strings.Split(k, "/")[len(strings.Split(k, "/"))-1]
		metadata = append(metadata, current_metadata)
	}

	splitter := textsplitter.NewMarkdownTextSplitter(textsplitter.WithModelName(openAIEmbeddingModel), textsplitter.WithChunkSize(chunkSize), textsplitter.WithChunkOverlap(chunkOverlap), textsplitter.WithHeadingHierarchy(true))
	docs, err := textsplitter.CreateDocuments(splitter, values, metadata)
	if err != nil {
		return nil, fmt.Errorf("\nFailed to chunk files: %w", err)
	}

	return docs, nil
}

// Embed and insert chunks as documents
func insertFiles(docs []schema.Document) error {
	client, _ := mongo.Connect(options.Client().ApplyURI(os.Getenv("SKUNKWORKS_ATLAS_URI")))

	defer func() {
		if err := client.Disconnect(context.Background()); err != nil {
			fmt.Errorf("\nFailed disconnecting the client: %w", err)
		}
	}()

	coll := client.Database(databaseName).Collection(collectionName)

	llm, err := openai.New(openai.WithBaseURL("https://skunkworks-gai-349.openai.azure.com/"), openai.WithModel(openAIEmbeddingModel), openai.WithEmbeddingModel(openAIEmbeddingModel), openai.WithAPIType(openai.APITypeAzure), openai.WithToken(os.Getenv("SKUNKWORKS_OPENAI_KEY")))
	if err != nil {
		return fmt.Errorf("\nFailed to create an embedder client: %w", err)
	}

	embedder, err := embeddings.NewEmbedder(llm)
	if err != nil {
		return fmt.Errorf("\nFailed to create an embedder: %w", err)
	}

	store := mongovector.New(coll, embedder, mongovector.WithIndex(indexName), mongovector.WithPath("embeddings"))
	coll.DeleteMany(context.Background(), nil)

	_, err = store.AddDocuments(context.Background(), docs)

	if err != nil {
		return fmt.Errorf("\nFailed adding documents: %w", err)
	}

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
