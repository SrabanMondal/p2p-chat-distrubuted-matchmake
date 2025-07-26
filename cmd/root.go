package cmd

import "github.com/spf13/cobra"

var rootCmd = &cobra.Command{
	Use:   "p2pchat",
	Short: "p2pchat cli provides secure way to talk to any random stranger or particular person using decentralized p2p system.",
}

func Execute() {
	cobra.CheckErr(rootCmd.Execute())
}

