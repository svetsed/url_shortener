package config

import (
	"fmt"
	"os"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	StoragePath string	 `yaml:"storage_path" env:"STORAGE_PATH" env-required:"true"`
	ShortURLLen int      `yaml:"short_url_len" env:"SHORT_URL_LEN" env-default:"8"`
    Env         string   `yaml:"env" env:"ENV" env-required:"true"`
    Server   	Server   `yaml:"server" env-prefix:"SERVER_"`
    Database 	Database `yaml:"database" env-prefix:"DATABASE_"`
}

type Server struct {
    Host    	string        `yaml:"host" env:"HOST" env-default:"localhost"`
    Port    	string        `yaml:"port" env:"PORT" env-default:"8080"`
    Timeout 	time.Duration `yaml:"timeout" env:"TIMEOUT" env-default:"10s"`
	IdleTimeout time.Duration `yaml:"idle_timeout" env:"IDLE_TIMEOUT" env-default:"60s"`
}

type Database struct {
    Host     string `yaml:"host" env:"HOST" env-default:"localhost"`
    Port     string `yaml:"port" env:"PORT" env-default:"5432"`
    User     string `yaml:"user" env:"USER" env-required:"true"`
    Password string `yaml:"password" env:"PASSWORD" env-required:"true"`
    Name     string `yaml:"name" env:"NAME" env-required:"true"`
    SSLMode  string `yaml:"sslmode" env:"SSLMODE" env-default:"disable"`
}

func LoadConfig() (*Config, error) {
	var cfg Config
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		err := cleanenv.ReadEnv(&cfg)
		if err != nil {
			return nil, fmt.Errorf("CONFIG_PATH is and environment variables not set")
		}
	}
	
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("config file does not exist: %s", configPath)
	}

	if err := cleanenv.ReadConfig(configPath, &cfg); err != nil {
		return nil, fmt.Errorf("can`t read config: %s", err)
	}

	// валидация конфига? длина короткой ссылки больше 4 
	// стораге паф существует

	return &cfg, nil
}

