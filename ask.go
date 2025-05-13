package mongospecgpt

import (
	"context"
	"fmt"
	"strings"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/schema"
	"github.com/tmc/langchaingo/vectorstores"
)

const defaultNumDocs = 4

type AskOptions struct {
	NumDocs    int
	PromptFunc PromptFunc
}

type AskOption func(*AskOptions)

func WithNumDocs(num int) AskOption {
	return func(opts *AskOptions) {
		opts.NumDocs = num
	}
}

func WithPromptFunc(promptFunc PromptFunc) AskOption {
	return func(opts *AskOptions) {
		opts.PromptFunc = promptFunc
	}
}

// Ask performs a full RAG query by embedding a uqer question, retrieving
// relevant `.md` spec chunks from a vector store, injecting those chumks into
// a prompt to an LLM (like GPT-4o) and then returning the final answer.
func Ask(
	ctx context.Context,
	store vectorstores.VectorStore,
	llm llms.Model,
	question string,
	opts ...AskOption,
) (string, error) {
	askOpts := &AskOptions{
		NumDocs:    defaultNumDocs,
		PromptFunc: DefaultPrompt,
	}

	for _, opt := range opts {
		opt(askOpts)
	}

	// Perform a similarity search on the vector store.
	results, err := store.SimilaritySearch(ctx, question, askOpts.NumDocs)
	if err != nil {
		return "", fmt.Errorf("failed to perform similarity search: %w", err)
	}

	prompt := askOpts.PromptFunc(question, results)

	resp, err := llm.GenerateContent(ctx, prompt)
	if err != nil {
		return "", fmt.Errorf("failed to call LLM: %w", err)
	}

	return strings.TrimSpace(resp.Choices[0].Content), nil
}

// PromptFunc is a function that takes a question and a list of chunks
// and returns a formatted string to be used as a prompt for an LLM.
type PromptFunc func(question string, chunks []schema.Document) []llms.MessageContent

var _ PromptFunc = DefaultPrompt

// DefaultPrompt is the default prompt function that formats the question to
// provide context to the LLM.
func DefaultPrompt(question string, chunks []schema.Document) []llms.MessageContent {
	var b strings.Builder

	b.WriteString("Context:\n")
	for _, doc := range chunks {
		b.WriteString(doc.PageContent)
		b.WriteString("\n---\n")
	}

	b.WriteString(fmt.Sprintf("\nQuestion: %s\n", question))

	return []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeSystem, "You are an expert on MongoDB specifications."),
		llms.TextParts(llms.ChatMessageTypeHuman, b.String()),
	}
}
