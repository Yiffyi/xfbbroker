package misc

import (
	"log/slog"
	"runtime/debug"

	"github.com/spf13/viper"
	"github.com/yiffyi/gorad"
)

func SetupLogger() {

	level := slog.LevelInfo
	if viper.GetBool("log.debug") {
		level = slog.LevelDebug
		dbginfo, _ := debug.ReadBuildInfo()
		println(dbginfo.String())
	}

	slog.SetDefault(slog.New(gorad.NewTextFileSlogHandler(viper.GetString("log.path"), level)))
	// stop := make(chan bool)

}
