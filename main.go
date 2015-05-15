package main

import (
	"fmt"
	"github.com/blasphemy/furry-robot/models"
	"github.com/blasphemy/furry-robot/utils"
	r "github.com/dancannon/gorethink"
	"github.com/go-martini/martini"
	"github.com/spf13/viper"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

var session *r.Session

func main() {
	ConfigInit()
	var err error
	session, err = r.Connect(r.ConnectOpts{
		Address: viper.GetString("RethinkDbConnectionString"),
	})
	if err != nil {
		log.Fatal(err.Error())
	}
	DatabaseInit()
	m := martini.Classic()
	m.Get("/", func(res http.ResponseWriter) {
		res.WriteHeader(http.StatusOK)
	})
	m.Get("/:id/:key", GetHandler)
	m.Get("/:id", GetHandler)
	m.Post("/api/up", postHandler)
	m.Run()
}

func GetHandler(p martini.Params, res http.ResponseWriter) {
	cur, err := r.Db(viper.GetString("DBName")).Table(viper.GetString("FileTable")).Get(p["id"]).Run(session)
	if err != nil {
		res.WriteHeader(http.StatusBadRequest)
		res.Write([]byte("Your meme was too dank for us"))
		return
	}
	var k models.File
	err = cur.One(&k)
	if err != nil {
		if err == r.ErrEmptyResult {
			res.WriteHeader(http.StatusNotFound)
			res.Write([]byte("Sorry, we couldn't find your meme"))
			return
		}
		res.WriteHeader(http.StatusBadRequest)
		res.Write([]byte("NOP STILL TOO DANK"))
		log.Println(err.Error())
		return
	}
	log.Println("retrieved file")
	if k.Private && p["key"] != k.AccessKey {
		res.WriteHeader(http.StatusForbidden)
		res.Write([]byte("You are not allowed to view this file"))
		return
	}
	log.Printf("Updating timestamp for file %s", k.Id)
	k.LastAccessed = time.Now()
	k.Hits++
	err = r.Db(viper.GetString("DBName")).Table(viper.GetString("FileTable")).Get(p["id"]).Update(&k).Exec(session)
	if err != nil {
		log.Printf("Could not update timestamp for file %s: %s", k.Id, err.Error())
	}
	err = nil
	cur, err = r.Db(viper.GetString("DBName")).Table(viper.GetString("FilePieceTable")).Filter(map[string]interface{}{"ParentId": p["id"]}).OrderBy("Seq").Run(session)
	if err != nil {
		log.Println(err.Error())
		res.WriteHeader(http.StatusBadRequest)
		res.Write([]byte("still too dank"))
	}
	var result models.FilePiece
	res.Header().Add("content-disposition", fmt.Sprintf(`inline; filename="%s"`, k.FileName))
	res.Header().Add("Content-Length", fmt.Sprintf("%d", k.FileSize))
	res.WriteHeader(http.StatusOK)
	for cur.Next(&result) {
		res.Write(result.Data)
	}
}

func postHandler(req *http.Request, res http.ResponseWriter) {
	var userFinished bool
	var FileFinished bool
	var file models.File
	//Generate a new ID on every request. Yes, this is kind of dumb, but we can get it out of the way here.
	newid, err := GetNewID()
	if strings.ToLower(newid) == "api" {
		newid, err = GetNewID()
	}
	if err != nil {
		res.WriteHeader(http.StatusInternalServerError)
		res.Write([]byte(err.Error()))
		return
	}
	log.Printf("New file id: %s", newid)
	file.Id = newid
	var user models.User
	var apikey string
	reader, err := req.MultipartReader()
	if err != nil {
		res.WriteHeader(http.StatusInternalServerError)
		res.Write([]byte(err.Error()))
		log.Println("Error opening reader")
		log.Println(err.Error())
		return
	}
	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			res.WriteHeader(http.StatusInternalServerError)
			res.Write([]byte(err.Error()))
			log.Println("NextPart error")
			log.Println(err.Error())
			return
		}
		switch part.FormName() {
		case "k":
			buffer := make([]byte, 1024)
			numread, err := part.Read(buffer)
			if err != nil {
				res.WriteHeader(http.StatusInternalServerError)
				res.Write([]byte(err.Error()))
				log.Println("K part.Read error")
				log.Println(err.Error())
				return
			}
			apikey = string(buffer[:numread])
			log.Printf("User Api key: %s", apikey)
			user, err = models.GetUserByApiKey(apikey, session, viper.GetString("DBName"), viper.GetString("UserTable"), viper.GetBool("Debug"))
			if !user.Active {
				res.WriteHeader(http.StatusUnauthorized)
				res.Write([]byte("You are not authorized to upload"))
				return
			}
			log.Printf("Updating user %s last activity.", user.Email)
			err = user.UpdateLastActivity(session, viper.GetString("DBName"), viper.GetString("UserTable"))
			if err != nil {
				log.Println(err.Error())
				err = nil //Not fatal
			}
			userFinished = true
		case "private":
			buffer := make([]byte, 1024)
			numread, err := part.Read(buffer)
			if err != nil {
				res.WriteHeader(http.StatusInternalServerError)
				res.Write([]byte(err.Error()))
				return
			}
			if string(buffer[:numread]) == "true" {
				file.Private = true
			}
		case "f":
			//This assumes that "f" will be the last parameter sent.
			//This is a stupid assumption
			if part.FileName() == "" {
				res.WriteHeader(http.StatusBadRequest)
				return
			}
			file.FileName = part.FileName()
			var Seq int64
			for {
				buffer := make([]byte, 1024)
				Read, err := part.Read(buffer)
				if err != nil {
					if err == io.EOF {
						FileFinished = true
						break
					}
					res.WriteHeader(http.StatusInternalServerError)
					res.Write([]byte(err.Error()))
					return
				}
				file.FileSize = file.FileSize + int64(Read)
				f := models.FilePiece{Data: buffer[:Read], Seq: Seq, ParentId: file.Id}
				err = r.Db(viper.GetString("DBName")).Table(viper.GetString("FilePieceTable")).Insert(&f).Exec(session)
				if err != nil {
					res.WriteHeader(http.StatusInternalServerError)
					res.Write([]byte(err.Error()))
					r.Db(viper.GetString("DBName")).Table(viper.GetString("FilePieceTable")).Filter(map[string]interface{}{"ParentId": file.Id}).Delete().Exec(session)
					return
				}
				Seq++
			}
		}
	}
	//Finally out of that mess.
	if !userFinished {
		res.WriteHeader(http.StatusBadRequest)
		res.Write([]byte("Please provide an API key"))
		return
	}
	if !FileFinished {
		res.WriteHeader(http.StatusBadRequest)
		res.Write([]byte("You did not provide a file"))
		return
	}
	file.AccessKey = utils.Base62Rand(viper.GetInt("AccessKeyLength"))
	err = r.Db(viper.GetString("DBName")).Table(viper.GetString("FileTable")).Insert(&file).Exec(session)
	if err != nil {
		res.WriteHeader(http.StatusInternalServerError)
		res.Write([]byte(err.Error()))
		log.Println("Error inserting file. Attempting to roll back")
		log.Println(err.Error())
		err = nil
		err = r.Db(viper.GetString("DBName")).Table(viper.GetString("FilePieceTable")).Filter(map[string]interface{}{"ParentId": file.Id}).Delete().Exec(session)
		if err != nil {
			log.Println("Error rolling back. Nothing to do here.")
			log.Println(err.Error())
		}
	}
	res.WriteHeader(http.StatusOK)
	res.Write([]byte("," + file.GetUrl(viper.GetString("BaseUrl")) + ","))
}

func GetNewID() (string, error) {
	err := r.Db(viper.GetString("DBName")).Table(viper.GetString("MetaTable")).Get("counter").Update(map[string]interface{}{"value": r.Row.Field("value").Add(1)}).Exec(session)
	if err != nil {
		log.Println(err.Error())
		return "", err
	}
	cursor, err := r.Db(viper.GetString("DBName")).Table(viper.GetString("MetaTable")).Get("counter").Field("value").Run(session)
	if err != nil {
		err := r.Db(viper.GetString("DBName")).Table(viper.GetString("MetaTable")).Get("counter").Exec(session)
		if err != r.ErrEmptyResult && err != nil {
			log.Println(err.Error())
			return "", err
		}
		err2 := r.Db(viper.GetString("DBName")).Table(viper.GetString("MetaTable")).Insert(map[string]interface{}{"id": "counter", "value": 0}).Exec(session)
		if err2 != nil {
			return "", err
		}
		return utils.Base62Encode(0), nil
	}
	var target uint64
	cursor.One(&target)
	if cursor.Err() != nil {
		return "", cursor.Err()
	}
	return utils.Base62Encode(target), nil
}
