package main

import (
	"fmt"

	mongospecgpt "github.com/prestonvasquez/mongo-spec-gpt"
	"github.com/spf13/cobra"
)

func runAsk(cmd *cobra.Command, args []string) error {
	_, err := mongospecgpt.Ask(cmd.Context(), args[0])
	if err != nil {
		return fmt.Errorf("failed to ask: %w", err)
	}

	return nil
}

func newAskCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ask",
		Short: "Ask a question about the MongoDB spec files",
	}

	cmd.Run = func(cmd *cobra.Command, args []string) {
		if err := runAsk(cmd, args); err != nil {
			cmd.PrintErrln(err)
		}
	}

	return cmd
}
