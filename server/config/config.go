package config

import (
	"fmt"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

func Read() *viper.Viper {
	// any approach to require this configuration into your program.

	pflag.String("port", ":1459", "Port to bind server to")
	pflag.String("db", "", "Path to your iCloud db (typically ~/Library/Messages/chat.db)")
	pflag.String("token", "UNIQUE_UUID_OR_OTHER_TOKEN", "Token to validate requests against")
	pflag.String("recheckInterval", "20000", "Interval in milliseconds to check for album updates")
	pflag.Parse()
	viper.BindPFlags(pflag.CommandLine)

	newConfig := viper.New()
	newConfig.SetConfigName("config") // name of config file (without extension)
	newConfig.SetConfigType("yaml")   // REQUIRED if the config file does not have the extension in the name
	newConfig.AddConfigPath(".")      // optionally look for config in the working directory
	newConfig.SetDefault("db", "")
	newConfig.SetDefault("port", 1459)
	newConfig.SetDefault("recheckInterval", 20000)
	newConfig.SetDefault("token", "UNIQUE_UUID_OR_OTHER_TOKEN")

	err := newConfig.ReadInConfig() // Find and read the config file
	if err != nil {                 // Handle errors reading the config file
		panic(fmt.Errorf("Fatal error config file: %s", err))
	}

	return newConfig
}
