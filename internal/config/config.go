package config

import (
	"flag"
	"fmt"
	"net"
	"net/url"
	"strings"

	"github.com/caarlos0/env/v11"
)

type Config struct {
	LoadAddress string	`env:"SERVER_ADDRESS" envDefault:":8080"`
	BaseAddress string	`env:"BASE_URL" envDefault:"http://localhost:8080"`
}

func NewDefaultConfig() *Config {
	return &Config{
		LoadAddress: ":8080",
		BaseAddress: "http://localhost:8080",
	}
}

func SettingConfig(cfg *Config) error {
	flag.StringVar(&cfg.LoadAddress, "a", ":8080", "address and port to run server")
	flag.StringVar(&cfg.BaseAddress, "b", "http://localhost:8080", "base address for the resulting shortened URL")

	if !flag.Parsed() {
		flag.Parse()
	}

	err := env.Parse(cfg)
	if err != nil {
		return fmt.Errorf("error parse env: %v", err)
	}

	if err := cfg.validate(); err != nil {
		return fmt.Errorf("invalid configuration: %v", err)
	}

	return nil
}

func (c *Config) validate() error {
	if err := c.validateLoadAddress(); err != nil {
		return fmt.Errorf("invalid load address: %w", err)
	}

	if err := c.validateBaseAddress(); err != nil {
        return fmt.Errorf("invalid base address: %w", err)
    }
    
    return nil
}

func (c *Config) validateLoadAddress() error {
	if c.LoadAddress == "" {
        return fmt.Errorf("address cannot be empty")
    }

	_, port, err := net.SplitHostPort(c.LoadAddress)
	if err != nil {
		return fmt.Errorf("invalid format (should be host:port): %w", err)
	}

	if port == "" {
		return fmt.Errorf("port is required")
	}

	return nil
}

func (c *Config) validateBaseAddress() error {
	if c.BaseAddress == "" {
        return fmt.Errorf("base address cannot be empty")
    }

	if !strings.HasPrefix(c.BaseAddress, "http://") && !strings.HasPrefix(c.BaseAddress, "https://") {
		c.BaseAddress = "http://" + c.BaseAddress
	}

	parsed, err := url.Parse(c.BaseAddress)
	if err != nil {
		return fmt.Errorf("invalid URL format: %w", err)
    }

	if parsed.Host == "" {
		return fmt.Errorf("host is required in base address")
    }

	host, port, _ := net.SplitHostPort(parsed.Host)
	if host == "" {
		return fmt.Errorf("invalid host in base address")
	}

	if port == "" {
		if parsed.Scheme == "https" {
			c.BaseAddress = strings.Replace(c.BaseAddress, parsed.Host, parsed.Host+":443", 1)
		} else {
			c.BaseAddress = strings.Replace(c.BaseAddress, parsed.Host, parsed.Host+":80", 1)
		}
	}

	return nil
}