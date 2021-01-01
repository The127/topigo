package config

import (
	"fmt"
	"github.com/kelseyhightower/envconfig"
	"gopkg.in/go-playground/validator.v9"
	"gopkg.in/yaml.v2"
	"log"
	"os"
)

type Config struct {
	Server struct {
		Host string `yaml:"host" envconfig:"SERVER__HOST" validate:"required,hostname"`
		Port int16  `yaml:"port" envconfig:"SERVER__PORT"`
	} `yaml:"server"`
	Storage struct {
		Directory             string `yaml:"directory" envconfig:"STORAGE__DIRECTORY" validate:"required,dir"`
		RetentionDays         int    `yaml:"retentionDays" envconfig:"STORAGE__RETENTION_DAYS" validate:"required,gt=1"`
		DeletionBufferDays    int    `yaml:"deletionBufferDays" envconfig:"STORAGE__DELETION_BUFFER_DAYS" validate:"required,gt=1"`
		DeletionIntervalHours int    `yaml:"deletionIntervalHours" envconfig:"STORAGE__DELETION_INTERVAL_HOURS" validate:"required,gt=1,lt=24"`
	} `yaml:"storage"`
}

func MakeConfig(configFilePath string, configEnvPrefix string) Config {
	var cfg Config

	readConfigFile(&cfg, configFilePath)
	readConfigEnv(&cfg, configFilePath)

	validate := validator.New()
	err := validate.Struct(cfg)

	if err != nil {
		for _, e := range err.(validator.ValidationErrors) {
			fmt.Errorf("%v", e)
		}
		os.Exit(1)
	}

	return cfg
}

func readConfigFile(cfg *Config, configFilePath string) {
	f, err := os.Open(configFilePath)
	if err != nil {
		log.Fatalf("error while opening config file (%v): %v", configFilePath, err)
	}
	defer f.Close()

	decoder := yaml.NewDecoder(f)
	err = decoder.Decode(cfg)
	if err != nil {
		log.Fatalf("error while processing configuration file (%v): %v", configFilePath, err)
	}
}

func readConfigEnv(cfg *Config, configEnvPrefix string) {
	err := envconfig.Process(configEnvPrefix, cfg)
	if err != nil {
		log.Fatalf("error while processing environment variables for configuration: %v", err)
	}
}
