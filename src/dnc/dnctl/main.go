package main

import (
	"dnc/dnctl/cmd"
)

func main() {

	rootCmd := cmd.NewRootCmd()
	rootCmd.AddCommand(cmd.ConfigCmd())
	rootCmd.AddCommand(cmd.RegisterCmd())
	rootCmd.AddCommand(cmd.UnregisterCmd())
	rootCmd.AddCommand(cmd.CreateCmd())
	rootCmd.AddCommand(cmd.DeleteCmd())
	rootCmd.AddCommand(cmd.GetCmd())

	rootCmd.Execute()
}
