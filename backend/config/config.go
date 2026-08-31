package config

import (
	"log"

	"github.com/spf13/viper"
)

type ServerConfig struct {
	Port string
	Mode string
}

type DatabaseConfig struct {
	DSN string
}

type JWTConfig struct {
	Secret      string
	ExpireHours int
}

type OCPPConfig struct {
	WSAddr string
}

type VDV261Config struct {
	Enable      bool   `mapstructure:"enable"       json:"enable"`
	NetworkMode string `mapstructure:"network_mode" json:"network_mode"`
	ListenAddr  string `mapstructure:"listen_addr"  json:"listen_addr"`
	URLPath     string `mapstructure:"url_path"     json:"url_path"`
	CertFile    string `mapstructure:"cert_file"    json:"cert_file"`
	KeyFile     string `mapstructure:"key_file"     json:"key_file"`
	RootCA      string `mapstructure:"root_ca"      json:"root_ca"`
}

type CertSigningConfig struct {
	DefaultDays         int      `mapstructure:"default_days"`
	DefaultMD           string   `mapstructure:"default_md"`
	IsCA                bool     `mapstructure:"is_ca"`
	KeyUsage            []string `mapstructure:"key_usage"`
	ExtendedKeyUsage    []string `mapstructure:"extended_key_usage"`
	AuthorityInfoAccess struct {
		CAIssuers string `mapstructure:"ca_issuers"`
		OCSP      string `mapstructure:"ocsp"`
	} `mapstructure:"authority_info_access"`
}

type Config struct {
	Server      ServerConfig
	Database    DatabaseConfig
	JWT         JWTConfig
	OCPP        OCPPConfig
	CertSigning CertSigningConfig `mapstructure:"certSigning"`
	VDV261      VDV261Config      `mapstructure:"vdv261"`
}

var Cfg Config

func Load() {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./config")
	viper.AddConfigPath(".")

	viper.SetDefault("server.port", "8080")
	viper.SetDefault("server.mode", "debug")
	viper.SetDefault("jwt.expireHours", 24)
	viper.SetDefault("ocpp.wsAddr", ":9101")
	viper.SetDefault("vdv261.enable", false)
	viper.SetDefault("vdv261.network_mode", "ipv6")
	viper.SetDefault("vdv261.listen_addr", ":9443")
	viper.SetDefault("vdv261.url_path", "/vdv")

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		log.Println("[config] no config file found, using env/defaults")
	}

	if err := viper.Unmarshal(&Cfg); err != nil {
		log.Fatalf("[config] unmarshal error: %v", err)
	}
}
