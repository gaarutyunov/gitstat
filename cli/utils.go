package cli

import (
	"github.com/spf13/cobra"
	"os"
	"strings"
)

func bindToEnv(cmd *cobra.Command, prefix string, flags ...string) error {
	for _, flag := range flags {
		if v, _ := cmd.PersistentFlags().GetString(flag); v == "" {
			err := cmd.PersistentFlags().Set(flag, os.Getenv(strings.ToUpper(prefix)+"_"+strings.ToUpper(flag)))
			if err != nil {
				return err
			}
		}
	}

	return nil
}
