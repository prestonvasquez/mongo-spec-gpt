package main

import "github.com/spf13/cobra"

func runAsk(cmd *cobra.Command, args []string) error {
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
