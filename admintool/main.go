package main

import (
	"errors"
	"github.com/blasphemy/furry-robot/models"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	r "gopkg.in/dancannon/gorethink.v1"
	"log"
	"net/http"
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
		err = r.DB(viper.GetString("DBName")).Table(viper.GetString("UserTable")).Insert(NewUser).Exec(session)
		if err != nil {
			log.Fatal(err.Error())
		}
		log.Printf("User %s API KEY: %s", NewUser.Email, NewUser.ApiKey)
	},
}

var Migrations = &cobra.Command{
	Use: "migrate",
	Run: func(cmd *cobra.Command, args []string) {
		RunDataBaseMigrations()
	},
}

func main() {
	ConfigInit()
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
	AdminTool.AddCommand(Migrations)
	AdminTool.Execute()
}

func ConfigInit() {
	viper.SetConfigName("furryrobot")
	viper.AddConfigPath("../")
	viper.SetDefault("RethinkDbConnectionString", "192.168.1.100:28015")
	viper.SetDefault("DBName", "FurryRobot")
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

func RunDataBaseMigrations() {
	err := FileSizeMigration()
	if err != nil {
		log.Fatal(err.Error())
	}
	err = MimeTypeMigration()
	if err != nil {
		log.Fatal(err.Error())
	}
}

func FileSizeMigration() error {
	log.Println("Beginning migration to add file sizes")
	cur, err := r.DB(viper.GetString("DBName")).Table(viper.GetString("FileTable")).Run(session)
	if err != nil {
		return err
	}
	counter := 0
	result := &models.File{}
	for cur.Next(result) {
		if result.FileSize == 0 {
			var ByteCounter int64
			cur2, err := r.DB(viper.GetString("DBName")).Table(viper.GetString("FilePieceTable")).Filter(map[string]interface{}{"ParentId": result.Id}).Run(session)
			if err != nil {
				return err
			}
			res2 := &models.FilePiece{}
			for cur2.Next(res2) {
				ByteCounter = ByteCounter + int64(len(res2.Data))
			}
			log.Printf("File %s size: %d", result.Id, ByteCounter)
			result.FileSize = ByteCounter
			err = r.DB(viper.GetString("DBName")).Table(viper.GetString("FileTable")).Update(result).Exec(session)
			if err != nil {
				return err
			}
			counter++
		}
	}
	log.Printf("Updated %d files", counter)
	return nil
}

func MimeTypeMigration() error {
	log.Println("Beginning migration to add mime types")
	cur, err := r.DB(viper.GetString("DBName")).Table(viper.GetString("FileTable")).Run(session)
	if err != nil {
		return err
	}
	counter := 0
	result := &models.File{}
	for cur.Next(result) {
		if result.MimeType == "" {
			cur2, err := r.DB(viper.GetString("DBName")).Table(viper.GetString("FilePieceTable")).Filter(map[string]interface{}{"ParentId": result.Id, "Seq": 0}).Run(session)
			if err != nil {
				return err
			}
			res2 := &models.FilePiece{}
			err = cur2.One(res2)
			if err != nil {
				return err
			}
			result.MimeType = http.DetectContentType(res2.Data)
			err = r.DB(viper.GetString("DBName")).Table(viper.GetString("FileTable")).Update(result).Exec(session)
			if err != nil {
				return err
			}
			counter++
		}
	}
	log.Printf("Updated %d files", counter)
	return nil
}
