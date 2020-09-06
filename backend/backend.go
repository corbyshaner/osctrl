package backend

import (
	"fmt"
	"time"

	"github.com/spf13/viper"

	"github.com/jinzhu/gorm"
	// Import mysql dialect
	_ "github.com/jinzhu/gorm/dialects/mysql"
)

const (
	// DBString to format connection string to database
	DBString = "%s:%s@(%s)/%s?charset=utf8mb4&parseTime=True&loc=Local"
	// DBDialect for the database to use
	DBDialect = "mysql"
	// DBKey to identify the configuration JSON key
	DBKey = "db"
)

// JSONConfigurationDB to hold all backend configuration values
type JSONConfigurationDB struct {
	Host            string `json:"host"`
	Port            string `json:"port"`
	Name            string `json:"name"`
	Username        string `json:"username"`
	Password        string `json:"password"`
	MaxIdleConns    int    `json:"max_idle_conns"`
	MaxOpenConns    int    `json:"max_open_conns"`
	ConnMaxLifetime int    `json:"conn_max_lifetime"`
}

// LoadConfiguration to load the DB configuration file and assign to variables
func LoadConfiguration(file, key string) (JSONConfigurationDB, error) {
	var config JSONConfigurationDB
	// Load file and read config
	viper.SetConfigFile(file)
	if err := viper.ReadInConfig(); err != nil {
		return config, err
	}
	// Backend values
	dbRaw := viper.Sub(key)
	if err := dbRaw.Unmarshal(&config); err != nil {
		return config, err
	}
	// No errors!
	return config, nil
}

// GetDB to get Mysql DB using GORM
func GetDB(config JSONConfigurationDB) (*gorm.DB, error) {
	// Generate DB connection string
	mysqlDSN := fmt.Sprintf(DBString, config.Username, config.Password, config.Host, config.Name)
	// Connect to DB
	db, err := gorm.Open(DBDialect, mysqlDSN)
	if err != nil {
		return nil, err
	}
	// Performance settings for DB access
	db.DB().SetMaxIdleConns(config.MaxIdleConns)
	db.DB().SetMaxOpenConns(config.MaxOpenConns)
	db.DB().SetConnMaxLifetime(time.Second * time.Duration(config.ConnMaxLifetime))

	return db, nil
}
