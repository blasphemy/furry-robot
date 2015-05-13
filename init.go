package main

import (
	"github.com/blasphemy/furry-robot/rethinkdbutils"
	"github.com/spf13/viper"
	"log"
)

func ConfigInit() {
	viper.SetConfigName("puush")
	viper.AddConfigPath(".")
	viper.SetDefault("RethinkDbConnectionString", "127.0.0.1:28015")
	viper.SetDefault("DBName", "puush")
	viper.SetDefault("FileTable", "Files")
	viper.SetDefault("FilePieceTable", "FilePieces")
	viper.SetDefault("MetaTable", "Meta")
	viper.SetDefault("UserTable", "Users")
	viper.SetDefault("BaseUrl", "http://127.0.0.1:3000/")
	viper.SetDefault("AccessKeyLength", 5)
	log.Println("Reading config")
	err := viper.ReadInConfig()
	if err != nil {
		log.Println("error reading config, using defaults")
		log.Println(err.Error())
	}
}

func DatabaseInit() {
	var err error
	err = rethinkdbutils.MakeDbIfNotExist(viper.GetString("DBName"), session)
	if err != nil {
		log.Fatal(err.Error())
	}
	err = rethinkdbutils.MakeTableIfNotExist(viper.GetString("DBName"), viper.GetString("FileTable"), session)
	if err != nil {
		log.Fatal(err.Error())
	}
	err = rethinkdbutils.MakeTableIfNotExist(viper.GetString("DBName"), viper.GetString("FilePieceTable"), session)
	if err != nil {
		log.Fatal(err.Error())
	}
	err = rethinkdbutils.MakeTableIfNotExist(viper.GetString("DBName"), viper.GetString("MetaTable"), session)
	if err != nil {
		log.Fatal(err.Error())
	}
	err = rethinkdbutils.MakeIndexIfNotExist(viper.GetString("DBName"), viper.GetString("FilePieceTable"), "ParentId", session)
	if err != nil {
		log.Fatal(err.Error())
	}
	err = rethinkdbutils.MakeTableIfNotExist(viper.GetString("DBName"), viper.GetString("UserTable"), session)
	if err != nil {
		log.Fatal(err.Error())
	}
	err = rethinkdbutils.MakeIndexIfNotExist(viper.GetString("DBName"), viper.GetString("FileTable"), "UserId", session)
	if err != nil {
		log.Fatal(err.Error())
	}
	err = rethinkdbutils.MakeIndexIfNotExist(viper.GetString("DBName"), viper.GetString("UserTable"), "ApiKey", session)
	if err != nil {
		log.Fatal(err.Error())
	}
}
