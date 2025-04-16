package cmd

import (
	"log/slog"
	"net/http"
	"runtime/debug"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/yiffyi/gorad"
	"github.com/yiffyi/xfbbroker/data"
	"github.com/yiffyi/xfbbroker/server"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start background loop and serve HTTP requests",
	Run: func(cmd *cobra.Command, args []string) {
		slog.Warn("Program started")

		level := slog.LevelInfo
		if viper.GetBool("log.debug") {
			level = slog.LevelDebug
			dbginfo, _ := debug.ReadBuildInfo()
			println(dbginfo.String())
		}

		slog.SetDefault(slog.New(gorad.NewTextFileSlogHandler(viper.GetString("log.path"), level)))
		// stop := make(chan bool)

		db := data.OpenDatabase(viper.GetString("db.dsn"))

		bg := server.CreateBackgroundLoop(db)
		bg.Start()

		if viper.GetBool("http.listen_tls") {
			http.ListenAndServeTLS(viper.GetString("http.listen_addr"), viper.GetString("http.tls_cert"), viper.GetString("http.tls_key"), server.CreateApiServer(db))
		} else {
			http.ListenAndServe(viper.GetString("http.listen_addr"), server.CreateApiServer(db))
		}
	},
}

func SetupServeCommand() *cobra.Command {
	return serveCmd
}
