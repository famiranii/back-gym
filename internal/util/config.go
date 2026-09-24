package util

import (
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	APP_NAME                  string        `json:"app_name"`
	APP_PORT                  string        `json:"app_port"`
	APP_DEBUG                 string        `json:"app_debug"`
	DB_CONNECTION             string        `json:"db_connection"`
	DB_HOST                   string        `json:"db_host"`
	DB_PORT                   string        `json:"db_port"`
	DB_USERNAME               string        `json:"db_username"`
	DB_PASSWORD               string        `json:"db_password"`
	DB_DATABASE               string        `json:"db_database"`
	MIGRATION_URL             string        `json:"migration_url"`
	JWT_SECRET                string        `mapstructure:"JWT_SECRET"`
	ACCESS_TOKEN_DURATION     time.Duration `mapstructure:"ACCESS_TOKEN_DURATION"`
	REFRESH_TOKEN_DURATION    time.Duration `mapstructure:"REFRESH_TOKEN_DURATION"`
	SMS_USERNAME              string        `mapstructure:"SMS_USERNAME"`
	SMS_PASSWORD              string        `mapstructure:"SMS_PASSWORD"`
	SMS_FROM                  string        `mapstructure:"SMS_FROM"`
	MAPIR_API_KEY             string        `mapstructure:"MAPIR_API_KEY"`
	MAPIR_DAILY_REQUEST_LIMIT int           `mapstructure:"MAPIR_DAILY_REQUEST_LIMIT"`
	COOKIE_DOMAIN             string        `mapstructure:"COOKIE_DOMAIN"`
	ENVIRONMENT               string        `mapstructure:"ENVIRONMENT"`
}

func LoadConfig(path string) (config Config, err error) {
	viper.AddConfigPath(path)
	viper.SetConfigName("app")
	viper.SetConfigType("env")

	viper.AutomaticEnv()

	viper.BindEnv("JWT_SECRET")
	viper.BindEnv("ACCESS_TOKEN_DURATION")
	viper.BindEnv("REFRESH_TOKEN_DURATION")
	viper.BindEnv("SMS_USERNAME")
	viper.BindEnv("SMS_PASSWORD")
	viper.BindEnv("SMS_FROM")
	viper.BindEnv("MAPIR_API_KEY")
	viper.BindEnv("MAPIR_DAILY_REQUEST_LIMIT")
	viper.BindEnv("COOKIE_DOMAIN")
	viper.BindEnv("ENVIRONMENT")
	err = viper.ReadInConfig()
	if err != nil {
		return
	}

	err = viper.Unmarshal(&config)
	return
}
