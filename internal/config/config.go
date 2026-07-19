package config

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	JWT      JWTConfig
	SMTP     SMTPConfig
	Feishu   FeishuConfig
}

type ServerConfig struct {
	Host string
	Port int
}

type DatabaseConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
	SSLMode  string
}

type JWTConfig struct {
	Secret      string
	ExpireHours int
}

const DefaultJWTSecret = "change-me-in-production"

func (j JWTConfig) UsesDefaultSecret() bool {
	return strings.TrimSpace(j.Secret) == "" || j.Secret == DefaultJWTSecret
}

type SMTPConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
}

type FeishuConfig struct {
	WebhookURL string
}

var AppConfig *Config

func loadEnvFile() {
	f, err := os.Open(".env")
	if err != nil {
		return
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])
		if os.Getenv(key) == "" {
			os.Setenv(key, val)
		}
	}
}

func Load() error {
	loadEnvFile()

	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./config")
	viper.AddConfigPath("/etc/uptime_ng")

	viper.SetEnvPrefix("UPTIME_NG")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	viper.SetDefault("server.host", "0.0.0.0")
	viper.SetDefault("server.port", 3000)
	viper.SetDefault("database.host", "localhost")
	viper.SetDefault("database.port", 5432)
	viper.SetDefault("database.user", "uptime")
	viper.SetDefault("database.password", "uptime123")
	viper.SetDefault("database.dbname", "uptime_ng")
	viper.SetDefault("database.sslmode", "disable")
	viper.SetDefault("jwt.secret", DefaultJWTSecret)
	viper.SetDefault("jwt.expirehours", 72)
	viper.SetDefault("smtp.host", "")
	viper.SetDefault("smtp.port", 587)
	viper.SetDefault("smtp.username", "")
	viper.SetDefault("smtp.password", "")
	viper.SetDefault("smtp.from", "")
	viper.SetDefault("feishu.webhook_url", "")

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return err
		}
	}

	AppConfig = &Config{
		Server: ServerConfig{
			Host: viper.GetString("server.host"),
			Port: viper.GetInt("server.port"),
		},
		Database: DatabaseConfig{
			Host:     viper.GetString("database.host"),
			Port:     viper.GetInt("database.port"),
			User:     viper.GetString("database.user"),
			Password: viper.GetString("database.password"),
			DBName:   viper.GetString("database.dbname"),
			SSLMode:  viper.GetString("database.sslmode"),
		},
		JWT: JWTConfig{
			Secret:      viper.GetString("jwt.secret"),
			ExpireHours: viper.GetInt("jwt.expirehours"),
		},
		SMTP: SMTPConfig{
			Host:     viper.GetString("smtp.host"),
			Port:     viper.GetInt("smtp.port"),
			Username: viper.GetString("smtp.username"),
			Password: viper.GetString("smtp.password"),
			From:     viper.GetString("smtp.from"),
		},
		Feishu: FeishuConfig{
			WebhookURL: viper.GetString("feishu.webhook_url"),
		},
	}

	return nil
}

func (d *DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s TimeZone=Asia/Shanghai",
		d.Host, d.Port, d.User, d.Password, d.DBName, d.SSLMode,
	)
}
