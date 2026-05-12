// Package config handles loading and validating application configuration
// from YAML files and CLI flags.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/viper"
)

// Config is the root configuration structure.
type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	Auth     AuthConfig     `mapstructure:"auth"`
	Storage  StorageConfig  `mapstructure:"storage"`
	TFTP     TFTPConfig     `mapstructure:"tftp"`
	Services ServicesConfig `mapstructure:"services"`
	LDAP     LDAPConfig     `mapstructure:"ldap"`
	Log      LogConfig      `mapstructure:"log"`
	FOS      FOSConfig      `mapstructure:"fos"`
}

type ServerConfig struct {
	HTTP        string `mapstructure:"http"`
	HTTPS       string `mapstructure:"https"`
	TLSCert     string `mapstructure:"tls_cert"`
	TLSKey      string `mapstructure:"tls_key"`
	BaseURL     string `mapstructure:"base_url"`
	// ReadHeaderTimeout limits only the time to read request headers, protecting
	// against slowloris attacks. ReadTimeout / WriteTimeout are intentionally not
	// set so that long-running image upload/download streams are never killed.
	ReadHeaderTimeout time.Duration `mapstructure:"read_header_timeout"`
}

type DatabaseConfig struct {
	DSN             string        `mapstructure:"dsn"`
	Host            string        `mapstructure:"host"`
	Port            int           `mapstructure:"port"`
	Name            string        `mapstructure:"name"`
	User            string        `mapstructure:"user"`
	Password        string        `mapstructure:"password"`
	SSLMode         string        `mapstructure:"sslmode"`
	MaxOpenConns    int           `mapstructure:"max_open_conns"`
	MaxIdleConns    int           `mapstructure:"max_idle_conns"`
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime"`
}

// DSNString builds a PostgreSQL connection string from individual fields
// if DSN is not explicitly set.
func (d DatabaseConfig) DSNString() string {
	if d.DSN != "" {
		return d.DSN
	}
	return fmt.Sprintf(
		"host=%s port=%d dbname=%s user=%s password=%s sslmode=%s",
		d.Host, d.Port, d.Name, d.User, d.Password, d.SSLMode,
	)
}

type AuthConfig struct {
	// JWTSecret is used to sign access, refresh, and boot tokens.
	JWTSecret          string        `mapstructure:"jwt_secret"`
	AccessTokenExpiry  time.Duration `mapstructure:"access_token_expiry"`
	RefreshTokenExpiry time.Duration `mapstructure:"refresh_token_expiry"`
	// BootTokenExpiry controls how long a boot session token is valid.
	// Default is 2 hours — long enough for slow-link imaging sessions.
	BootTokenExpiry    time.Duration `mapstructure:"boot_token_expiry"`
}

type StorageConfig struct {
	// BasePath is the root path where images are stored.
	BasePath       string `mapstructure:"base_path"`
	SnapinPath     string `mapstructure:"snapin_path"`
	KernelPath     string `mapstructure:"kernel_path"`
	MaxUploadBytes int64  `mapstructure:"max_upload_bytes"`
}

type TFTPConfig struct {
	Enabled   bool   `mapstructure:"enabled"`
	Listen    string `mapstructure:"listen"`
	RootDir   string `mapstructure:"root_dir"`
}

type ServicesConfig struct {
	SchedulerInterval   time.Duration `mapstructure:"scheduler_interval"`
	ReplicatorInterval        time.Duration `mapstructure:"replicator_interval"`
	SnapinReplicatorInterval  time.Duration `mapstructure:"snapin_replicator_interval"`
	PingInterval              time.Duration `mapstructure:"ping_interval"`
	ImageSizeInterval         time.Duration `mapstructure:"image_size_interval"`
	SnapinHashInterval        time.Duration `mapstructure:"snapin_hash_interval"`
	MulticastInterval         time.Duration `mapstructure:"multicast_interval"`
	// TaskTimeout is how long a queued task waits for the host to handshake
	// before being auto-cancelled. Zero disables (default 30 minutes).
	TaskTimeout time.Duration `mapstructure:"task_timeout"`
}

type LDAPConfig struct {
	Enabled      bool   `mapstructure:"enabled"`
	URL          string `mapstructure:"url"`
	BindDN       string `mapstructure:"bind_dn"`
	BindPassword string `mapstructure:"bind_password"`
	BaseDN       string `mapstructure:"base_dn"`
	UserFilter   string `mapstructure:"user_filter"`
}

type LogConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"` // "json" or "text"
}

// FOSConfig controls automatic download of fos-next kernel and initramfs
// artifacts during `fog install`.
type FOSConfig struct {
	// ReleaseURL is the base URL of the fos-next release to download from.
	// The installer appends the individual file names (KernelFile, InitFile,
	// sha256sums) to this URL.
	// Default: https://github.com/nemvince/fos-next/releases/latest/download
	ReleaseURL string `mapstructure:"release_url"`
	// KernelFile is the filename of the kernel image in the release archive.
	KernelFile string `mapstructure:"kernel_file"`
	// InitFile is the filename of the compressed initramfs.
	InitFile string `mapstructure:"init_file"`
	// SkipDownload disables automatic downloading during `fog install`.
	// Set to true if you manage the kernel files manually.
	SkipDownload bool `mapstructure:"skip_download"`
}

// Load reads configuration from the given file path (or the default search
// paths when filePath is empty). Settings not present in the file fall back
// to built-in defaults.
func Load(filePath string) (*Config, error) {
	v := viper.New()
	setDefaults(v)

	if filePath != "" {
		v.SetConfigFile(filePath)
		if err := v.ReadInConfig(); err != nil {
			return nil, fmt.Errorf("reading config file %q: %w", filePath, err)
		}
	} else {
		v.SetConfigName("config")
		v.SetConfigType("yaml")
		v.AddConfigPath("/etc/fog")
		v.AddConfigPath("$HOME/.fog")
		v.AddConfigPath(".")
		_ = v.ReadInConfig() // not found is fine — defaults cover it
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshalling config: %w", err)
	}

	return &cfg, nil
}

func setDefaults(v *viper.Viper) {
	v.SetDefault("server.http", ":80")
	v.SetDefault("server.https", ":443")
	v.SetDefault("server.base_url", "http://localhost")
	v.SetDefault("server.read_header_timeout", 10*time.Second)

	v.SetDefault("database.host", "localhost")
	v.SetDefault("database.port", 5432)
	v.SetDefault("database.name", "fog")
	v.SetDefault("database.user", "fog")
	v.SetDefault("database.password", "fog")
	v.SetDefault("database.sslmode", "disable")
	v.SetDefault("database.max_open_conns", 25)
	v.SetDefault("database.max_idle_conns", 5)
	v.SetDefault("database.conn_max_lifetime", 5*time.Minute)

	v.SetDefault("auth.access_token_expiry", 15*time.Minute)
	v.SetDefault("auth.refresh_token_expiry", 7*24*time.Hour)

	v.SetDefault("storage.base_path", "/opt/fog/images")
	v.SetDefault("storage.snapin_path", "/opt/fog/snapins")
	v.SetDefault("storage.kernel_path", "/var/www/html/fog/service/ipxe")
	v.SetDefault("storage.max_upload_bytes", int64(100<<30)) // 100 GiB

	v.SetDefault("tftp.enabled", true)
	v.SetDefault("tftp.listen", ":69")
	v.SetDefault("tftp.root_dir", "/tftpboot")

	v.SetDefault("services.scheduler_interval", 60*time.Second)
	v.SetDefault("services.replicator_interval", 10*time.Minute)
	v.SetDefault("services.snapin_replicator_interval", 10*time.Minute)
	v.SetDefault("services.ping_interval", 5*time.Minute)
	v.SetDefault("services.image_size_interval", 1*time.Hour)
	v.SetDefault("services.snapin_hash_interval", 30*time.Minute)
	v.SetDefault("services.multicast_interval", 10*time.Second)
	v.SetDefault("services.task_timeout", 30*time.Minute)

	v.SetDefault("log.level", "info")
	v.SetDefault("log.format", "text")

	v.SetDefault("fos.release_url", "https://github.com/nemvince/fos-next/releases/latest/download")
	v.SetDefault("fos.kernel_file", "bzImage")
	v.SetDefault("fos.init_file", "init.xz")
	v.SetDefault("fos.skip_download", false)
}

// Defaults returns a Config populated entirely from defaults (no file needed).
func Defaults() *Config {
	cfg, _ := Load("")
	return cfg
}

// WriteDefault writes cfg as YAML to the given path, creating parent
// directories as needed.
func WriteDefault(cfg *Config, path string) error {
	v := viper.New()
	setDefaults(v)
	v.Set("database.host", cfg.Database.Host)
	v.Set("database.port", cfg.Database.Port)
	v.Set("database.name", cfg.Database.Name)
	v.Set("database.user", cfg.Database.User)
	v.Set("database.password", cfg.Database.Password)
	v.Set("server.http", cfg.Server.HTTP)
	v.Set("storage.base_path", cfg.Storage.BasePath)
	v.Set("storage.snapin_path", cfg.Storage.SnapinPath)
	v.Set("auth.jwt_secret", cfg.Auth.JWTSecret)
	v.SetConfigFile(path)
	v.SetConfigType("yaml")

	// Ensure parent directory exists.
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", filepath.Dir(path), err)
	}
	return v.WriteConfigAs(path)
}
