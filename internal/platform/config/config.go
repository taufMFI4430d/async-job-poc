package config

import (
	"errors"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
)

const (
	defaultAppEnvironment = "development"
	defaultHTTPAddress    = ":8080"
	defaultLogLevel       = "info"
	defaultMySQLPort      = 3306
	defaultRedisPort      = 6379
	defaultRedisDB        = 0
	defaultRedisQueueName = "jobs:pending"
)

type Config struct {
	App   AppConfig
	HTTP  HTTPConfig
	MySQL MySQLConfig
	Redis RedisConfig
	Log   LogConfig
}

type AppConfig struct {
	Environment string
}

type HTTPConfig struct {
	Address string
}

type MySQLConfig struct {
	Host     string
	Port     int
	Database string
	User     string
	Password string
}

type RedisConfig struct {
	Host      string
	Port      int
	Password  string
	DB        int
	QueueName string
}

type LogConfig struct {
	Level string
}

func Load() (Config, error) {
	mysqlPort, err := intFromEnvironment("MYSQL_PORT", defaultMySQLPort)
	if err != nil {
		return Config{}, err
	}

	redisPort, err := intFromEnvironment("REDIS_PORT", defaultRedisPort)
	if err != nil {
		return Config{}, err
	}

	redisDB, err := intFromEnvironment("REDIS_DB", defaultRedisDB)
	if err != nil {
		return Config{}, err
	}

	cfg := Config{
		App: AppConfig{
			Environment: valueOrDefault(
				"APP_ENV",
				defaultAppEnvironment,
			),
		},
		HTTP: HTTPConfig{
			Address: valueOrDefault(
				"HTTP_ADDRESS",
				defaultHTTPAddress,
			),
		},
		MySQL: MySQLConfig{
			Host:     os.Getenv("MYSQL_HOST"),
			Port:     mysqlPort,
			Database: os.Getenv("MYSQL_DATABASE"),
			User:     os.Getenv("MYSQL_USER"),
			Password: os.Getenv("MYSQL_PASSWORD"),
		},
		Redis: RedisConfig{
			Host:     os.Getenv("REDIS_HOST"),
			Port:     redisPort,
			Password: os.Getenv("REDIS_PASSWORD"),
			DB:       redisDB,
			QueueName: valueOrDefault(
				"REDIS_QUEUE_NAME",
				defaultRedisQueueName,
			),
		},
		Log: LogConfig{
			Level: valueOrDefault(
				"LOG_LEVEL",
				defaultLogLevel,
			),
		},
	}

	if err := cfg.Validate(); err != nil {
		return Config{}, fmt.Errorf("validate configuration: %w", err)
	}

	return cfg, nil
}

func (cfg Config) Validate() error {
	var validationErrors []error

	requiredValues := map[string]string{
		"MYSQL_HOST":     cfg.MySQL.Host,
		"MYSQL_DATABASE": cfg.MySQL.Database,
		"MYSQL_USER":     cfg.MySQL.User,
		"MYSQL_PASSWORD": cfg.MySQL.Password,
		"REDIS_HOST":     cfg.Redis.Host,
	}

	for name, value := range requiredValues {
		if strings.TrimSpace(value) == "" {
			validationErrors = append(
				validationErrors,
				fmt.Errorf("%s is required", name),
			)
		}
	}

	if err := validatePort("MYSQL_PORT", cfg.MySQL.Port); err != nil {
		validationErrors = append(validationErrors, err)
	}

	if err := validatePort("REDIS_PORT", cfg.Redis.Port); err != nil {
		validationErrors = append(validationErrors, err)
	}

	if cfg.Redis.DB < 0 {
		validationErrors = append(
			validationErrors,
			fmt.Errorf("REDIS_DB must be zero or greater"),
		)
	}

	if strings.TrimSpace(cfg.Redis.QueueName) == "" {
		validationErrors = append(
			validationErrors,
			fmt.Errorf("REDIS_QUEUE_NAME is required"),
		)
	}

	if err := validateHTTPAddress(cfg.HTTP.Address); err != nil {
		validationErrors = append(validationErrors, err)
	}

	if !isSupportedEnvironment(cfg.App.Environment) {
		validationErrors = append(
			validationErrors,
			fmt.Errorf(
				"APP_ENV must be one of development, test, staging, or production",
			),
		)
	}

	if !isSupportedLogLevel(cfg.Log.Level) {
		validationErrors = append(
			validationErrors,
			fmt.Errorf(
				"LOG_LEVEL must be one of debug, info, warn, or error",
			),
		)
	}

	return errors.Join(validationErrors...)
}

func valueOrDefault(name, defaultValue string) string {
	value, exists := os.LookupEnv(name)
	if !exists || strings.TrimSpace(value) == "" {
		return defaultValue
	}

	return strings.TrimSpace(value)
}

func intFromEnvironment(name string, defaultValue int) (int, error) {
	rawValue, exists := os.LookupEnv(name)
	if !exists || strings.TrimSpace(rawValue) == "" {
		return defaultValue, nil
	}

	value, err := strconv.Atoi(rawValue)
	if err != nil {
		return 0, fmt.Errorf("%s must be a valid integer: %w", name, err)
	}

	return value, nil
}

func validatePort(name string, port int) error {
	if port < 1 || port > 65535 {
		return fmt.Errorf("%s must be between 1 and 65535", name)
	}

	return nil
}

func validateHTTPAddress(address string) error {
	_, portValue, err := net.SplitHostPort(address)
	if err != nil {
		return fmt.Errorf(
			"HTTP_ADDRESS must have a host:port format such as :8080: %w",
			err,
		)
	}

	port, err := strconv.Atoi(portValue)
	if err != nil {
		return fmt.Errorf("HTTP_ADDRESS contains an invalid port: %w", err)
	}

	return validatePort("HTTP_ADDRESS port", port)
}

func isSupportedEnvironment(environment string) bool {
	switch strings.ToLower(environment) {
	case "development", "test", "staging", "production":
		return true
	default:
		return false
	}
}

func isSupportedLogLevel(level string) bool {
	switch strings.ToLower(level) {
	case "debug", "info", "warn", "error":
		return true
	default:
		return false
	}
}
