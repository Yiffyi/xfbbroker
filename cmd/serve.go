package cmd

import (
	"log/slog"
	"net/http"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/yiffyi/xfbbroker/data"
	"github.com/yiffyi/xfbbroker/server"
)

var serveNoLooper *bool

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start background loop and serve HTTP requests",
	Run: func(cmd *cobra.Command, args []string) {
		slog.Warn("Program started")
		db := data.OpenDatabase(viper.GetString("db.dsn"))

		if *serveNoLooper {
			slog.Warn("BackgroundLoop not started")
		} else {
			bg := server.CreateBackgroundLoop(db)
			bg.Start()
		}

		if viper.GetBool("http.listen_tls") {
			http.ListenAndServeTLS(viper.GetString("http.listen_addr"), viper.GetString("http.tls_cert"), viper.GetString("http.tls_key"), server.CreateApiServer(db))
		} else {
			http.ListenAndServe(viper.GetString("http.listen_addr"), server.CreateApiServer(db))
		}
	},
}

func SetupServeCommand() *cobra.Command {
	serveNoLooper = serveCmd.Flags().Bool("no-looper", false, "disable BackgroundLoop")
	return serveCmd
}
