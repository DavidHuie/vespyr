package vespyr_test

import (
	"fmt"
	"os"

	"github.com/spf13/viper"
)

func init() {
	viper.SetConfigFile("../../test-config.yml")
	if err := viper.ReadInConfig(); err != nil {
		fmt.Println("Can't read config:", err)
		os.Exit(1)
	}
}
