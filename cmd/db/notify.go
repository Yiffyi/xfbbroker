package db

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/yiffyi/xfbbroker/data"
)

var notifyCmd = &cobra.Command{
	Use:       "notify",
	Short:     "Edit notification channels",
	Args:      cobra.OnlyValidArgs,
	ValidArgs: []string{"new", "ls", "rm"},
	Run: func(cmd *cobra.Command, args []string) {
		db := data.OpenDatabase(viper.GetString("db.dsn"))
		op := args[0]
		switch op {
		case "ls":
			var cs []data.NotificationChannel
			err := db.Find(&cs).Error
			if err != nil {
				panic(err)
			}

			for _, c := range cs {
				fmt.Printf("%+v\n", c)
			}
		case "new":
			userId, _ := cmd.Flags().GetUint("user-id")
			name, _ := cmd.Flags().GetString("name")
			enabled, _ := cmd.Flags().GetBool("enable")
			t, _ := cmd.Flags().GetString("type")
			p, _ := cmd.Flags().GetString("param")
			_, err := db.CreateNotificationChannel(userId, name, t, p, enabled)
			if err != nil {
				panic(err)
			}
		case "rm":
			id, _ := cmd.Flags().GetUint("id")
			err := db.Delete(&data.NotificationChannel{}, id).Error
			if err != nil {
				panic(err)
			}
		}

	},
}

func SetupNotifyCommand() *cobra.Command {
	notifyCmd.Flags().UintP("id", "i", 0, "Specify ID")
	notifyCmd.Flags().UintP("user-id", "u", 0, "Specify user ID")
	notifyCmd.Flags().StringP("name", "n", "", "Specify name of the notification channel")
	notifyCmd.Flags().Bool("enable", true, "Whether this notification channel is enabled")
	notifyCmd.Flags().StringP("type", "t", "", "Specify channel type")
	notifyCmd.Flags().StringP("param", "p", "", "Specify channel parameter")

	return notifyCmd
}
