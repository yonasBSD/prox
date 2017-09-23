package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cmd = &cobra.Command{
	Use:   "prox",
	Short: "A process runner for Procfile-based applications",
	Run:   run,
}

func main() {
	cmd.PersistentFlags().BoolP("verbose", "v", false, "enable detailed log output for debugging")

	viper.AutomaticEnv()
	viper.SetEnvPrefix("PROX")
	viper.BindPFlags(cmd.PersistentFlags())

	if err := cmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
