package db

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/yiffyi/xfbbroker/data"
	"github.com/yiffyi/xfbbroker/xfb"
)

var userId *uint
var userSessionId *string

var userCmd = &cobra.Command{
	Use:       "user",
	Short:     "Manage users in database",
	Args:      cobra.OnlyValidArgs,
	ValidArgs: []string{"new", "ls", "rm", "codepay"},
	Run: func(cmd *cobra.Command, args []string) {
		db := data.OpenDatabase(viper.GetString("db.dsn"))
		op := args[0]
		switch op {
		case "ls":
			{
				var users []data.User
				err := db.Find(&users).Error
				if err != nil {
					panic(err)
				}

				for _, u := range users {
					fmt.Printf("%+v\n", u)
				}
			}
		case "new":
			{
				data, _, err := xfb.GetUserDefaultLoginInfo(*userSessionId)
				if err != nil {
					panic(err)
				}
				fmt.Printf("%v", data)

				u, err := db.CreateUser(data.UserName, data.Openid, *userSessionId, data.ID)
				if err != nil {
					panic(err)
				}

				if v, _ := cmd.Flags().GetBool("auto-topup"); v {
					u.EnableAutoTopup = true
				}

				if v, _ := cmd.Flags().GetBool("trans-notify"); v {
					u.EnableTransNotify = true
				}

				err = db.UpdateUserLoopEnabled(u)
				if err != nil {
					panic(err)
				}

			}
		case "rm":
			{
				if *userId == 0 {
					panic("You must specify a user ID to delete")
				}
				err := db.Delete(&data.User{}, userId).Error
				if err != nil {
					panic(err)
				}
			}
		}
		return
	},
}

func SetupUserCommand() *cobra.Command {
	userId = userCmd.Flags().UintP("id", "i", 0, "Specify ID")
	userSessionId = userCmd.Flags().StringP("session-id", "s", "", "Specify sessionId for user")
	userCmd.Flags().Bool("auto-topup", false, "Enable auto top-up")
	userCmd.Flags().Bool("trans-notify", false, "Enable transaction notification")
	return userCmd
}
