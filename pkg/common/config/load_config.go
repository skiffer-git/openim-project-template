package config

import (
	"github.com/mitchellh/mapstructure"
	"github.com/openimsdk/tools/errs"
	"github.com/spf13/viper"
	"os"
	"path/filepath"
	"strings"
)

func Load(configDirectory string, configFileName string, envPrefix string, config any) error {
	if os.Getenv(DeploymentType) == KUBERNETES {
		mountPath := os.Getenv(MountConfigFilePath)
		if mountPath == "" {
			return errs.ErrArgs.WrapMsg(MountConfigFilePath + " env is empty")
		}
		return loadConfigK8s(mountPath, configFileName, config)
	}
	return loadConfig(filepath.Join(configDirectory, configFileName), envPrefix, config)
}

func loadConfig(path string, envPrefix string, config any) error {
	v := viper.New()
	v.SetConfigFile(path)
	v.SetEnvPrefix(envPrefix)
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	if err := v.ReadInConfig(); err != nil {
		return errs.WrapMsg(err, "failed to read config file", "path", path, "envPrefix", envPrefix)
	}

	if err := v.Unmarshal(config, func(config *mapstructure.DecoderConfig) {
		config.TagName = "mapstructure"
	}); err != nil {
		return errs.WrapMsg(err, "failed to unmarshal config", "path", path, "envPrefix", envPrefix)
	}
	return nil
}

func loadConfigK8s(mountPath string, configFileName string, config any) error {
	configFilePath := filepath.Join(mountPath, configFileName)
	v := viper.New()
	v.SetConfigFile(configFilePath)

	if err := v.ReadInConfig(); err != nil {
		return errs.WrapMsg(err, "failed to read config file", "path", configFilePath)
	}

	if err := v.Unmarshal(config, func(config *mapstructure.DecoderConfig) {
		config.TagName = "mapstructure"
	}); err != nil {
		return errs.WrapMsg(err, "failed to unmarshal config", "path", configFilePath)
	}
	return nil
}
