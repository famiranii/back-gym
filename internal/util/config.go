package util

import "github.com/spf13/viper"

type Config struct {
	APP_NAME      string `json:"app_name"`
	APP_PORT      string `json:"app_port"`
	APP_DEBUG     string `json:"app_debug"`
	DB_CONNECTION string `json:"db_connection"`
	DB_HOST       string `json:"db_host"`
	DB_PORT       string `json:"db_port"`
	DB_USERNAME   string `json:"db_username"`
	DB_PASSWORD   string `json:"db_password"`
	DB_DATABASE   string `json:"db_database"`
	MIGRATION_URL string `json:"migration_url"`
}

func LoadConfig(path string) (config Config, err error) {
	viper.AddConfigPath(path)
	viper.SetConfigName("app")
	viper.SetConfigType("env")
	viper.AutomaticEnv()
	err = viper.ReadInConfig()
	if err != nil {
		return
	}
	err = viper.Unmarshal(&config)
	return
}
