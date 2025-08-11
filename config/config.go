package config

import (
	"fmt"
	"go-ask/internal/debug"
	"os"
	"path/filepath"
	"github.com/spf13/viper"
)

func Init(cfgFile string) error {
	if cfgFile == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		cfgFile = filepath.Join(home, "/.config/ask/config.yaml")
	}

	viper.SetConfigFile(cfgFile)

	if err := viper.ReadInConfig(); err != nil {
		return fmt.Errorf("[!] Unable to read config: %w", err)
	}

	debug.Print("[+] Config loaded:%s", viper.ConfigFileUsed())
	return nil
}
