package apiengine

import (
	"fmt"

	"github.com/spf13/viper"
)

// Config provides the ability to read and write configuration files
type Config = *viper.Viper

var (
	c *viper.Viper
)

func InitializeConfig(configName string) {
	viper.SetConfigFile(configName)
	viper.AddConfigPath(".")

	if err := viper.ReadInConfig(); err != nil {
		//only log error if file was specified
		if configName != "" {
			fmt.Println(err)
		}
	}
	c = viper.GetViper()
}

func MergeNewConfig(configName string) {
	InitializeConfig(configName)
}

// GetConfig provides the config singleton
func GetConfig() Config {
	if c == nil {
		InitializeConfig("")
		return c
	}
	return c
}
