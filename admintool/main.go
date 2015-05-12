package main

import (
	"errors"
	"github.com/blasphemy/puush/models"
	r "github.com/dancannon/gorethink"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"log"
	"time"
)

var NewUser *models.User
var session *r.Session

var AdminTool = &cobra.Command{
	Use:   "admintool",
	Short: "Admin tool",
	Run: func(cmd *cobra.Command, args []string) {
		log.Println("puush server admin tool")
	},
}

var AddUser = &cobra.Command{
	Use: "adduser",
	Run: func(cmd *cobra.Command, args []string) {
		log.Println("Adding user")
		uuid, err := GetNewApiKey()
		if err != nil {
			log.Fatal(err.Error())
		}
		NewUser.ApiKey = uuid
		NewUser.Epoch = time.Now()
		NewUser.LastActivity = time.Now()
		err = r.Db(viper.GetString("DBName")).Table(viper.GetString("UserTable")).Insert(NewUser).Exec(session)
		if err != nil {
			log.Fatal(err.Error())
		}
	},
}

func main() {
	SetViperJunk()
	var err error
	session, err = r.Connect(r.ConnectOpts{
		Address: viper.GetString("RethinkDbConnectionString"),
	})
	if err != nil {
		log.Fatal(err.Error())
	}
	NewUser = &models.User{}
	AddUser.Flags().StringVarP(&NewUser.Email, "email", "e", "", "users email address")
	AddUser.Flags().BoolVarP(&NewUser.Active, "active", "a", true, "user is active")
	AdminTool.AddCommand(AddUser)
	AdminTool.Execute()
}

func SetViperJunk() {
	viper.SetConfigName("puush")
	viper.AddConfigPath("../")
	viper.SetDefault("RethinkDbConnectionString", "127.0.0.1:28015")
	viper.SetDefault("DBName", "puush")
	viper.SetDefault("FileTable", "Files")
	viper.SetDefault("FilePieceTable", "FilePieces")
	viper.SetDefault("MetaTable", "Meta")
	viper.SetDefault("UserTable", "Users")
	viper.SetDefault("BaseUrl", "http://127.0.0.1:3000/")
	log.Println("Reading config")
	err := viper.ReadInConfig()
	if err != nil {
		log.Println("error reading config, using defaults")
		log.Println(err.Error())
	}
}

func GetNewApiKey() (string, error) {
	cur, err := r.UUID().Run(session)
	if err != nil {
		return "", err
	}
	var k interface{}
	err = cur.One(&k)
	if err != nil {
		return "", err
	}
	result, ok := k.(string)
	if !ok {
		return "", errors.New("Could not turn UUID into string")
	}
	return result, nil
}
