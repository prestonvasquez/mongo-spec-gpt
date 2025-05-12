package main

import "github.com/spf13/cobra"

func main() {
	cmd := &cobra.Command{
		Use:     "mongo-spec-gpt",
		Short:   "Query MongoDB Spec files using GPT and retrieval-augmented generation (RAG)",
		Version: "0.0.0-alpha",
	}

	cmd.AddCommand(newAskCommand())
	cmd.AddCommand(newSyncCommand())

	if err := cmd.Execute(); err != nil {
		cmd.PrintErrln(err)
	}
}
