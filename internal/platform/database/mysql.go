package database

import (
	"context"
	"errors"
	"fmt"
	"net"
	"time"

	driver "github.com/go-sql-driver/mysql"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

const (
	defaultMaxOpenConnections = 25
	defaultMaxIdleConnections = 10
	defaultConnectionLifetime = 30 * time.Minute
	defaultConnectionTimeout  = 5 * time.Second
)

type MySQLOptions struct {
	Host     string
	Port     int
	Database string
	User     string
	Password string
}

type MySQL struct {
	gormDB *gorm.DB
}

func OpenMySQL(ctx context.Context, options MySQLOptions) (*MySQL, error) {
	if err := validateMySQLOptions(options); err != nil {
		return nil, err
	}

	driverConfig := driver.Config{
		User:                 options.User,
		Passwd:               options.Password,
		Net:                  "tcp",
		Addr:                 net.JoinHostPort(options.Host, fmt.Sprintf("%d", options.Port)),
		DBName:               options.Database,
		Params:               map[string]string{"charset": "utf8mb4"},
		Collation:            "utf8mb4_0900_ai_ci",
		Loc:                  time.UTC,
		ParseTime:            true,
		Timeout:              defaultConnectionTimeout,
		ReadTimeout:          defaultConnectionTimeout,
		WriteTimeout:         defaultConnectionTimeout,
		AllowNativePasswords: true,
		CheckConnLiveness:    true,
	}

	gormDB, err := gorm.Open(
		mysql.New(mysql.Config{DSN: driverConfig.FormatDSN()}),
		&gorm.Config{
			Logger:         gormlogger.Default.LogMode(gormlogger.Silent),
			PrepareStmt:    true,
			TranslateError: true,
			NowFunc: func() time.Time {
				return time.Now().UTC()
			},
		},
	)
	if err != nil {
		return nil, fmt.Errorf("open MySQL connection: %w", err)
	}

	sqlDB, err := gormDB.DB()
	if err != nil {
		return nil, fmt.Errorf("access underlying MySQL connection: %w", err)
	}

	sqlDB.SetMaxOpenConns(defaultMaxOpenConnections)
	sqlDB.SetMaxIdleConns(defaultMaxIdleConnections)
	sqlDB.SetConnMaxLifetime(defaultConnectionLifetime)

	if err := sqlDB.PingContext(ctx); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("ping MySQL: %w", err)
	}

	return &MySQL{gormDB: gormDB}, nil
}

func (mysqlDB *MySQL) GORM() *gorm.DB {
	return mysqlDB.gormDB
}

func (mysqlDB *MySQL) Ping(ctx context.Context) error {
	if mysqlDB == nil || mysqlDB.gormDB == nil {
		return errors.New("MySQL connection is not initialized")
	}

	sqlDB, err := mysqlDB.gormDB.DB()
	if err != nil {
		return fmt.Errorf("access underlying MySQL connection: %w", err)
	}

	if err := sqlDB.PingContext(ctx); err != nil {
		return fmt.Errorf("ping MySQL: %w", err)
	}

	return nil
}

func (mysqlDB *MySQL) Close() error {
	if mysqlDB == nil || mysqlDB.gormDB == nil {
		return nil
	}

	sqlDB, err := mysqlDB.gormDB.DB()
	if err != nil {
		return fmt.Errorf("access underlying MySQL connection: %w", err)
	}

	if err := sqlDB.Close(); err != nil {
		return fmt.Errorf("close MySQL connection: %w", err)
	}

	return nil
}

func validateMySQLOptions(options MySQLOptions) error {
	if options.Host == "" {
		return errors.New("MySQL host is required")
	}

	if options.Port < 1 || options.Port > 65535 {
		return errors.New("MySQL port must be between 1 and 65535")
	}

	if options.Database == "" {
		return errors.New("MySQL database is required")
	}

	if options.User == "" {
		return errors.New("MySQL user is required")
	}

	if options.Password == "" {
		return errors.New("MySQL password is required")
	}

	return nil
}
