package main

import (
	"dnc/dnctl/cmd"
)

func main() {

	rootCmd := cmd.NewRootCmd()
	rootCmd.AddCommand(cmd.ConfigCmd())
	rootCmd.AddCommand(cmd.RegisterCmd())
	rootCmd.AddCommand(cmd.CreateCmd())

	rootCmd.Execute()
}
