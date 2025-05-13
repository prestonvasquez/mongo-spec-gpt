package main

import (
	"fmt"
	"os"

	mongospecgpt "github.com/prestonvasquez/mongo-spec-gpt"
	"github.com/prestonvasquez/mongo-spec-gpt/internal/mongoutil"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type askOptions struct {
	llmProvider string
}

func getLLM(provider string) (llms.Model, error) {
	switch provider {
	case "openai":
		return openai.New()
	default:
		return nil, fmt.Errorf("unsupported LLM provider: %s", provider)
	}
}

func runAsk(cmd *cobra.Command, args []string, opts askOptions) error {
	// Default the store to an Atlas Cluster. This can be generalized later.
	uri := os.Getenv("MONGODB_URI")
	if uri == "" {
		return fmt.Errorf("MONGODB_URI environment variable is not set")
	}

	logrus.Infof("connecting to MongoDB at %s", uri)

	client, err := mongo.Connect(options.Client().ApplyURI(uri))
	if err != nil {
		return fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	defer func() {
		if err := client.Disconnect(cmd.Context()); err != nil {
			panic(err)
		}
	}()

	// Ping the server.
	if err := client.Ping(cmd.Context(), nil); err != nil {
		return fmt.Errorf("failed to ping MongoDB: %w", err)
	}

	logrus.Info("connected to MongoDB")

	logrus.Info("creating vector store")

	store, err := mongoutil.Store(cmd.Context(), client)
	if err != nil {
		return fmt.Errorf("failed to create vector store: %w", err)
	}

	logrus.Info("vector store created")

	// Get the LLM from the provider.
	llm, err := getLLM(opts.llmProvider)
	if err != nil {
		return fmt.Errorf("failed to get LLM: %w", err)
	}

	logrus.Infof("using LLM provider: %s", opts.llmProvider)
	_, err = mongospecgpt.Ask(cmd.Context(), store, llm, args[0])
	if err != nil {
		return fmt.Errorf("failed to ask: %w", err)
	}

	logrus.Info("ask completed")

	return nil
}

func newAskCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ask",
		Short: "Ask a question about the MongoDB spec files",
	}

	opts := askOptions{}

	cmd.Flags().StringVarP(&opts.llmProvider, "llm-provider", "l", "openai", "LLM provider to use (e.g., OpenAI, Anthropic)")

	cmd.Run = func(cmd *cobra.Command, args []string) {
		if err := runAsk(cmd, args, opts); err != nil {
			cmd.PrintErrln(err)
		}
	}

	return cmd
}
