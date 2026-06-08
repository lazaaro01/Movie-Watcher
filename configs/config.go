package configs

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

type Config struct {
	Telegram    TelegramConfig `mapstructure:"telegram"`
	Monitor     MonitorConfig  `mapstructure:"monitor"`
	Database    DatabaseConfig `mapstructure:"database"`
	CityID      int            `mapstructure:"city_id"`
	TheaterID   int            `mapstructure:"theater_id"`
	Partnership string         `mapstructure:"partnership"`
}

type TelegramConfig struct {
	BotToken string `mapstructure:"bot_token"`
	ChatID   int64  `mapstructure:"chat_id"`
}

type MonitorConfig struct {
	IntervalMinutes int `mapstructure:"interval_minutes"`
}

type DatabaseConfig struct {
	Path string `mapstructure:"path"`
}

var AppConfig *Config

func Init() error {
	v := viper.New()

	v.SetConfigName("config")
	v.SetConfigType("json")

	configPaths := []string{
		".",
		filepath.Join(os.Getenv("HOME"), ".movie-watcher"),
		"/etc/movie-watcher",
	}

	for _, p := range configPaths {
		v.AddConfigPath(p)
	}

	v.SetDefault("telegram.bot_token", "")
	v.SetDefault("telegram.chat_id", 0)
	v.SetDefault("monitor.interval_minutes", 5)
	v.SetDefault("database.path", "./movie-watcher.db")
	v.SetDefault("city_id", 36)
	v.SetDefault("theater_id", 1446)
	v.SetDefault("partnership", "cinepolis")

	v.AutomaticEnv()
	v.SetEnvPrefix("MW")

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			AppConfig = &Config{
				Telegram:    TelegramConfig{},
				Monitor:     MonitorConfig{IntervalMinutes: 5},
				Database:    DatabaseConfig{Path: "./movie-watcher.db"},
				CityID:      36,
				TheaterID:   1446,
				Partnership: "cinepolis",
			}
			return nil
		}
		return fmt.Errorf("error reading config: %w", err)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return fmt.Errorf("error unmarshaling config: %w", err)
	}

	AppConfig = &cfg
	return nil
}

func GetConfigPath() string {
	return "config.json"
}

func WriteDefaultConfig() error {
	v := viper.New()
	v.SetConfigName("config")
	v.SetConfigType("json")

	v.SetDefault("telegram.bot_token", "")
	v.SetDefault("telegram.chat_id", 0)
	v.SetDefault("monitor.interval_minutes", 5)
	v.SetDefault("database.path", "./movie-watcher.db")
	v.SetDefault("city_id", 36)
	v.SetDefault("theater_id", 1446)
	v.SetDefault("partnership", "cinepolis")

	return v.WriteConfigAs("config.json")
}
