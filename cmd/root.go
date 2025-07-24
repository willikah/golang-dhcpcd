package cmd

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "golang-dhcpcd",
	Short: "golang-dhcpcd is a DHCP client daemon written in Go",
}

func Execute() {
	cobra.CheckErr(rootCmd.Execute())
}
