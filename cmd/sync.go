package main

import "github.com/spf13/cobra"

func runSync(cmd *cobra.Command, args []string) error {
	return nil
}

func newSyncCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Fetch .md specs and chunk them by heading, generate embeddings, and store them",
	}

	cmd.Run = func(cmd *cobra.Command, args []string) {
		if err := runAsk(cmd, args); err != nil {
			cmd.PrintErrln(err)
		}
	}

	return cmd
}
