package main

import (
	"bufio"
	"errors"
	"github.com/blasphemy/puush/models"
	"github.com/blasphemy/puush/rethinkdbutils"
	"github.com/blasphemy/urls/utils"
	r "github.com/dancannon/gorethink"
	"github.com/go-martini/martini"
	"github.com/spf13/viper"
	"log"
	"net/http"
	"time"
)

//var store map[string]File
var session *r.Session

var config models.Config

type File struct {
	Id       string `gorethink:"id"`
	UserId   string
	FileName string
	//Data         []byte
	Epoch        time.Time
	LastAccessed time.Time
	//Pieces       []FilePiece
}

type FilePiece struct {
	Seq      int64
	ParentId string
	Data     []byte
}

func main() {
	SetViperJunk()
	config = models.Config{
		RethinkDbConnectionString: viper.GetString("RethinkDbConnectionString"),
		Db:             viper.GetString("DBName"),
		FileTable:      viper.GetString("FileTable"),
		FilePieceTable: viper.GetString("FilePieceTable"),
		MetaTable:      viper.GetString("MetaTable"),
		UserTable:      viper.GetString("UserTable"),
		BaseUrl:        viper.GetString("BaseUrl"),
	}
	var err error
	session, err = r.Connect(r.ConnectOpts{
		Address: config.RethinkDbConnectionString,
	})
	if err != nil {
		log.Fatal(err.Error())
	}
	err = rethinkdbutils.MakeDbIfNotExist(config.Db, session)
	if err != nil {
		log.Fatal(err.Error())
	}
	err = rethinkdbutils.MakeTableIfNotExist(config.Db, config.FileTable, session)
	if err != nil {
		log.Fatal(err.Error())
	}
	err = rethinkdbutils.MakeTableIfNotExist(config.Db, config.FilePieceTable, session)
	if err != nil {
		log.Fatal(err.Error())
	}
	err = rethinkdbutils.MakeTableIfNotExist(config.Db, config.MetaTable, session)
	if err != nil {
		log.Fatal(err.Error())
	}
	err = rethinkdbutils.MakeIndexIfNotExist(config.Db, config.FilePieceTable, "ParentId", session)
	if err != nil {
		log.Fatal(err.Error())
	}
	err = rethinkdbutils.MakeTableIfNotExist(config.Db, config.UserTable, session)
	if err != nil {
		log.Fatal(err.Error())
	}
	err = rethinkdbutils.MakeIndexIfNotExist(config.Db, config.FileTable, "UserId", session)
	if err != nil {
		log.Fatal(err.Error())
	}
	err = rethinkdbutils.MakeIndexIfNotExist(config.Db, config.UserTable, "ApiKey", session)
	if err != nil {
		log.Fatal(err.Error())
	}
	//store = make(map[string]File)
	m := martini.Classic()
	m.Get("/", func(res http.ResponseWriter) {
		res.WriteHeader(http.StatusOK)
		/*
			for _, x := range store["1"].Pieces {
				res.Write(x.Data)
			}
		*/
	})
	m.Get("/:id", GetHandler)
	m.Post("/api/up", postHandler)
	m.Run()
}

func GetHandler(p martini.Params, res http.ResponseWriter) {
	cur, err := r.Db(config.Db).Table(config.FileTable).Get(p["id"]).Run(session)
	if err != nil {
		res.WriteHeader(http.StatusBadRequest)
		res.Write([]byte("Your meme was too dank for us"))
		return
	}
	var k File
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
	log.Printf("Updating timestamp for file %s", k.Id)
	k.LastAccessed = time.Now()
	err = r.Db(config.Db).Table(config.FileTable).Get(p["id"]).Update(&k).Exec(session)
	if err != nil {
		log.Printf("Could not update timestamp for file %s: %s", k.Id, err.Error())
	}
	err = nil
	cur, err = r.Db(config.Db).Table(config.FilePieceTable).Filter(map[string]interface{}{"ParentId": p["id"]}).OrderBy("Seq").Run(session)
	if err != nil {
		log.Println(err.Error())
		res.WriteHeader(http.StatusBadRequest)
		res.Write([]byte("still too dank"))
	}
	var result FilePiece
	res.WriteHeader(http.StatusOK)
	for cur.Next(&result) {
		res.Write(result.Data)
	}
}

func postHandler(req *http.Request, res http.ResponseWriter) {
	if req.FormValue("z") != "poop" {
		res.WriteHeader(http.StatusBadRequest)
		res.Write([]byte("Sorry, your meme wasn't dank enough."))
		return
	}
	file, header, err := req.FormFile("f")
	if err != nil {
		res.WriteHeader(http.StatusBadRequest)
		res.Write([]byte("Your meme was too dank for us"))
		return
	}
	cur, err := r.Db(config.Db).Table(config.UserTable).Filter(map[string]interface{}{"ApiKey": req.FormValue("k")}).Run(session)
	if err != nil {
		res.WriteHeader(http.StatusInternalServerError)
		res.Write([]byte("Your meme was too dank for us"))
		return
	}
	user := &models.User{}
	err = cur.One(user)
	if err != nil {
		if err == r.ErrEmptyResult {
			res.WriteHeader(http.StatusBadRequest)
			res.Write([]byte("Your user could not be found"))
			log.Println(err.Error())
			return
		}
		res.WriteHeader(http.StatusInternalServerError)
		res.Write([]byte("Your meme was too dank for us"))
		log.Println(err.Error())
		return
	}
	if !user.Active {
		res.WriteHeader(http.StatusUnauthorized)
		res.Write([]byte("lol ur b& haha #rekt"))
		return
	}
	newId, err := GetNewID()
	if err != nil {
		res.WriteHeader(http.StatusInternalServerError)
		res.Write([]byte("Your meme was too dank for us"))
		log.Println(err.Error())
		return
	}
	log.Printf("Updating user %s last activity", user.Email)
	err = user.UpdateLastActivity(session, config.Db, config.UserTable)
	if err != nil {
		log.Println(err.Error())
		err = nil //Not fatal
	}
	f := File{Id: newId, FileName: header.Filename, Epoch: time.Now(), LastAccessed: time.Now(), UserId: user.Id}
	err = r.Db(config.Db).Table(config.FileTable).Insert(&f).Exec(session)
	if err != nil {
		res.WriteHeader(http.StatusInternalServerError)
		res.Write([]byte("Your meme was too dank for us"))
		log.Println(err.Error())
		return
	}
	SeqCounter := int64(0)
	reader := bufio.NewReader(file)
	scanner := bufio.NewScanner(reader)
	scanner.Split(bufio.ScanBytes)
	ByteCounter := int64(0)
	currentPiece := FilePiece{Seq: SeqCounter, ParentId: f.Id}
	for scanner.Scan() {
		currentPiece.Data = append(currentPiece.Data, scanner.Bytes()...)
		ByteCounter++
		if ByteCounter >= 1024 {
			ByteCounter = 0
			SeqCounter++
			err = r.Db(config.Db).Table(config.FilePieceTable).Insert(&currentPiece).Exec(session)
			if err != nil {
				res.WriteHeader(http.StatusInternalServerError)
				res.Write([]byte("Your meme was too dank for us"))
				log.Println(err.Error())
				//try to undo whatever the fuck we just did before erroring
				err = nil
				err = r.Db(config.Db).Table(config.FileTable).Get(f.Id).Delete().Exec(session)
				if err != nil {
					log.Printf("Problem rolling back file %s: %s", f.Id, err.Error())
					err = nil
				}
				err = r.Db(config.Db).Table(config.FilePieceTable).Filter(map[string]interface{}{"ParentId": f.Id}).Delete().Exec(session)
				if err != nil {
					log.Printf("problem rolling back file pieces for %s: %s", f.Id, err.Error())
				}
				return
			}
			currentPiece = FilePiece{Seq: SeqCounter, ParentId: f.Id}
		}
	}
	err = r.Db(config.Db).Table(config.FilePieceTable).Insert(&currentPiece).Exec(session)
	if err != nil {
		res.WriteHeader(http.StatusInternalServerError)
		res.Write([]byte("Your meme was too dank for us"))
		log.Println(err.Error())
		//try to undo whatever the fuck we just did before erroring
		err = nil
		err = r.Db(config.Db).Table(config.FileTable).Get(f.Id).Delete().Exec(session)
		if err != nil {
			log.Printf("Problem rolling back file %s: %s", f.Id, err.Error())
			err = nil
		}
		err = r.Db(config.Db).Table(config.FilePieceTable).Filter(map[string]interface{}{"ParentId": f.Id}).Delete().Exec(session)
		if err != nil {
			log.Printf("problem rolling back file pieces for %s: %s", f.Id, err.Error())
		}
		return
	}
	res.WriteHeader(http.StatusOK)
	res.Write([]byte(config.BaseUrl + f.Id))
	res.Write([]byte("," + config.BaseUrl + f.Id + ","))
}

func GetNewID() (string, error) {
	var target interface{}
	err := r.Db(config.Db).Table(config.MetaTable).Get("counter").Update(map[string]interface{}{"value": r.Row.Field("value").Add(1)}).Exec(session)
	if err != nil {
		log.Println("EXEC")
		log.Println(err.Error())
		return "", err
	}
	cursor, err := r.Db(config.Db).Table(config.MetaTable).Get("counter").Field("value").Run(session)
	if err != nil {
		err := r.Db(config.Db).Table(config.MetaTable).Get("counter").Exec(session)
		if err != r.ErrEmptyResult && err != nil {
			log.Println("CURSOR")
			log.Println(err.Error())
			return "", err
		}
		err2 := r.Db(config.Db).Table(config.MetaTable).Insert(map[string]interface{}{"id": "counter", "value": 0}).Exec(session)
		if err2 != nil {
			return "", err
		}
		return "", nil
	}
	cursor.One(&target)
	if cursor.Err() != nil {
		log.Println("ONE")
		return "", cursor.Err()
	}
	final, ok := target.(float64)
	if !ok {
		return "", errors.New("Cannot convert counter to float64")
	}
	return utils.Base62Encode(uint64(final)), nil
}

func SetViperJunk() {
	viper.SetConfigName("puush")
	viper.AddConfigPath(".")
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
