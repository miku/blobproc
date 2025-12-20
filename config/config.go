package config

import (
	"fmt"
	"os"
	"path"
	"time"

	"github.com/adrg/xdg"
	"github.com/spf13/viper"
)

type Config struct {
	// Common settings
	Debug    bool          `mapstructure:"debug"`
	LogFile  string        `mapstructure:"log_file"`
	SpoolDir string        `mapstructure:"spool_dir"`
	Timeout  time.Duration `mapstructure:"timeout"`

	// S3 settings
	S3 S3Config `mapstructure:"s3"`

	// GROBID settings
	Grobid GrobidConfig `mapstructure:"grobid"`

	// Processing settings
	Processing ProcessingConfig `mapstructure:"processing"`
}

type S3Config struct {
	Endpoint      string `mapstructure:"endpoint"`
	AccessKey     string `mapstructure:"access_key"`
	SecretKey     string `mapstructure:"secret_key"`
	DefaultBucket string `mapstructure:"default_bucket"`
	UseSSL        bool   `mapstructure:"use_ssl"`
}

type GrobidConfig struct {
	Host        string        `mapstructure:"host"`
	MaxFileSize int64         `mapstructure:"max_file_size"`
	Timeout     time.Duration `mapstructure:"timeout"`
}

type ProcessingConfig struct {
	Workers   int  `mapstructure:"workers"`
	Parallel  bool `mapstructure:"parallel"`
	KeepSpool bool `mapstructure:"keep_spool"`
}

func Init() (*viper.Viper, error) {
	v := viper.New()

	// Set defaults
	setDefaults(v)

	// Config file search paths
	v.SetConfigName("blobproc")
	v.SetConfigType("yaml")
	v.AddConfigPath(".")
	v.AddConfigPath("$HOME/.config/blobproc")
	v.AddConfigPath("/etc/blobproc")

	// Environment variable prefix
	v.SetEnvPrefix("BLOBPROC")
	v.AutomaticEnv()

	// Read config file if exists
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			// If there's a config file but it's malformed, warn and continue with defaults
			fmt.Fprintf(os.Stderr, "Warning: error reading config file: %v (using defaults)\n", err)
		}
	}

	return v, nil
}

func setDefaults(v *viper.Viper) {
	// Common defaults
	v.SetDefault("debug", false)
	v.SetDefault("spool_dir", path.Join(xdg.DataHome, "blobproc", "spool"))
	v.SetDefault("timeout", "5m")

	// S3 defaults
	v.SetDefault("s3.endpoint", "localhost:9000")
	v.SetDefault("s3.access_key", "minioadmin")
	v.SetDefault("s3.secret_key", "minioadmin")
	v.SetDefault("s3.default_bucket", "sandcrawler")
	v.SetDefault("s3.use_ssl", false)

	// GROBID defaults
	v.SetDefault("grobid.host", "http://localhost:8070")
	v.SetDefault("grobid.max_file_size", 256*1024*1024) // 256MB
	v.SetDefault("grobid.timeout", "30s")

	// Processing defaults
	v.SetDefault("processing.workers", 4)
	v.SetDefault("processing.parallel", false)
	v.SetDefault("processing.keep_spool", false)
}
