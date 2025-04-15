package main

import (
	"runtime/debug"

	"log/slog"
	"net/http"

	"github.com/spf13/viper"
	"github.com/yiffyi/gorad"
	"github.com/yiffyi/xfbbroker/data"
	"github.com/yiffyi/xfbbroker/misc"
	"github.com/yiffyi/xfbbroker/server"
)

func main() {
	dbginfo, _ := debug.ReadBuildInfo()
	println(dbginfo.String())
	slog.Warn("Program started")

	err := misc.LoadConfig([]string{})
	if err != nil {
		panic(err)
	}

	level := slog.LevelInfo
	if viper.GetBool("log.debug") {
		level = slog.LevelDebug
	}
	slog.SetDefault(slog.New(gorad.NewTextFileSlogHandler(viper.GetString("log.path"), level)))
	// stop := make(chan bool)

	db := data.OpenDatabase(viper.GetString("db.dsn"))

	if viper.GetBool("http.listen_tls") {
		http.ListenAndServeTLS(viper.GetString("http.listen_addr"), viper.GetString("http.tls_cert"), viper.GetString("http.tls_key"), server.CreateApiServer(db))
	} else {
		http.ListenAndServe(viper.GetString("http.listen_addr"), server.CreateApiServer(db))
	}
}
