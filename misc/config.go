package misc

import (
	"github.com/spf13/viper"
)

const DefaultConfigName = "config"
const DefaultConfigType = "toml"

func setupViper(searchPaths []string) {
	viper.SetConfigName(DefaultConfigName)
	viper.SetConfigType(DefaultConfigType)

	for _, path := range searchPaths {
		viper.AddConfigPath(path)
	}

	viper.SetDefault("log.path", "xfbbroker.log")
	viper.SetDefault("log.console", true)
	viper.SetDefault("log.debug", true)

	viper.SetDefault("loop.check_transaction_interval", 10)
	viper.SetDefault("loop.check_balance_interval", 120)

	viper.SetDefault("http.listen_addr", ":8080")
	viper.SetDefault("http.listen_tls", false)
	viper.SetDefault("http.tls_cert", "cert.pem")
	viper.SetDefault("http.tls_key", "key.pem")
	viper.SetDefault("http.auth_endpoint", "http://localhost:8080/_/xfb/auth")
	viper.SetDefault("http.auth_callback", "https://webapp.xiaofubao.com@localhost:8443/_/xfb/auth?platform=WECHAT_H5&schoolCode=20090820")

	viper.SetDefault("db.dsn", "sqlite3://xfbbroker.db")
}

func LoadConfig(extraSearchPaths []string) error {
	setupViper(extraSearchPaths)

	viper.SafeWriteConfig()
	err := viper.ReadInConfig()
	return err
}
