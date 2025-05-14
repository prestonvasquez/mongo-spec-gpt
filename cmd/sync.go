package main

import (
	"fmt"

	mongospecgpt "github.com/prestonvasquez/mongo-spec-gpt"
	"github.com/spf13/cobra"
)

func runSync(cmd *cobra.Command, args []string, _ mongospecgpt.SyncOptions) error {
	err := mongospecgpt.Sync(cmd.Context())
	if err != nil {
		return fmt.Errorf("failed to sync: %w", err)
	}

	return nil
}

func newSyncCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Fetch .md specs and chunk them by heading, generate embeddings, and store them",
	}

	cmd.Run = func(cmd *cobra.Command, args []string) {
		if err := runSync(cmd, args, mongospecgpt.SyncOptions{}); err != nil {
			cmd.PrintErrln(err)
		}
	}

	return cmd
}
