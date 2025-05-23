package mongospecgpt

import (
	"context"
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"
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

	queries := strings.Split(question, "|")
	response := ""

	for _, query := range queries {
		// Perform a similarity search on the vector store.
		results, err := store.SimilaritySearch(ctx, query, askOpts.NumDocs)
		if err != nil {
			return "", fmt.Errorf("failed to perform similarity search: %w", err)
		}

		var prompt string

		// Chain previous response as context, if it exists
		if response != "" {
			prompt = askOpts.PromptFunc(query, results, response)
		} else {
			prompt = askOpts.PromptFunc(query, results)
		}

		logrus.Infof("Calling with this prompt: %s", prompt)

		response, err = llms.GenerateFromSinglePrompt(ctx, llm, prompt, llms.WithTemperature(1))
		if err != nil {
			return "", fmt.Errorf("failed to call LLM: %w", err)
		}
	}

	return response, nil
}

// PromptFunc is a function that takes a question, a list of chunks, and an optional previous response
// and returns a formatted string to be used as a prompt for an LLM.
type PromptFunc func(question string, chunks []schema.Document, chainedContext ...string) string

var _ PromptFunc = DefaultPrompt

// DefaultPrompt is the default prompt function that formats the question to
// provide context to the LLM.
func DefaultPrompt(question string, chunks []schema.Document, chainedContext ...string) string {
	var b strings.Builder
	b.WriteString("You are an expert on MongoDB specifications.\n\n")
	b.WriteString("Use the following context to answer the question accurately. Keep your answer grounded in this context. If you're unsure, say 'I don't know.'\n\n")
	b.WriteString("Context:\n")

	for i, doc := range chunks {
		source := doc.Metadata["source"]
		heading := doc.Metadata["heading"]
		b.WriteString(fmt.Sprintf("[Chunk %d: %s / %s]\n", i+1, source, heading))
		b.WriteString(doc.PageContent + "\n---\n")
	}

	if len(chainedContext) > 0 {
		b.WriteString(chainedContext[0])
	}

	b.WriteString(fmt.Sprintf("\nQuestion: %s\n", question))
	return b.String()
}
