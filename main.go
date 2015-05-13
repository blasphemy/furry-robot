package main

import (
	"bufio"
	"fmt"
	"github.com/blasphemy/furry-robot/models"
	"github.com/blasphemy/furry-robot/utils"
	r "github.com/dancannon/gorethink"
	"github.com/go-martini/martini"
	"github.com/spf13/viper"
	"log"
	"net/http"
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
	cur, err := r.Db(viper.GetString("DBName")).Table(viper.GetString("UserTable")).Filter(map[string]interface{}{"ApiKey": req.FormValue("k")}).Run(session)
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
	err = user.UpdateLastActivity(session, viper.GetString("DBName"), viper.GetString("UserTable"))
	if err != nil {
		log.Println(err.Error())
		err = nil //Not fatal
	}
	f := models.File{
		Id:           newId,
		FileName:     header.Filename,
		Epoch:        time.Now(),
		LastAccessed: time.Now(),
		UserId:       user.Id,
		Hits:         0,
		AccessKey:    utils.Base62Rand(viper.GetInt("AccessKeyLength")),
	}
	if req.FormValue("private") == "true" {
		f.Private = true
	}
	err = r.Db(viper.GetString("DBName")).Table(viper.GetString("FileTable")).Insert(&f).Exec(session)
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
	currentPiece := models.FilePiece{Seq: SeqCounter, ParentId: f.Id}
	for scanner.Scan() {
		currentPiece.Data = append(currentPiece.Data, scanner.Bytes()...)
		ByteCounter++
		if ByteCounter >= 1024 {
			ByteCounter = 0
			SeqCounter++
			err = r.Db(viper.GetString("DBName")).Table(viper.GetString("FilePieceTable")).Insert(&currentPiece).Exec(session)
			if err != nil {
				res.WriteHeader(http.StatusInternalServerError)
				res.Write([]byte("Your meme was too dank for us"))
				log.Println(err.Error())
				//try to undo whatever the fuck we just did before erroring
				err = nil
				err = r.Db(viper.GetString("DBName")).Table(viper.GetString("FileTable")).Get(f.Id).Delete().Exec(session)
				if err != nil {
					log.Printf("Problem rolling back file %s: %s", f.Id, err.Error())
					err = nil
				}
				err = r.Db(viper.GetString("DBName")).Table(viper.GetString("FilePieceTable")).Filter(map[string]interface{}{"ParentId": f.Id}).Delete().Exec(session)
				if err != nil {
					log.Printf("problem rolling back file pieces for %s: %s", f.Id, err.Error())
				}
				return
			}
			currentPiece = models.FilePiece{Seq: SeqCounter, ParentId: f.Id}
		}
	}
	err = r.Db(viper.GetString("DBName")).Table(viper.GetString("FilePieceTable")).Insert(&currentPiece).Exec(session)
	if err != nil {
		res.WriteHeader(http.StatusInternalServerError)
		res.Write([]byte("Your meme was too dank for us"))
		log.Println(err.Error())
		//try to undo whatever the fuck we just did before erroring
		err = nil
		err = r.Db(viper.GetString("DBName")).Table(viper.GetString("FileTable")).Get(f.Id).Delete().Exec(session)
		if err != nil {
			log.Printf("Problem rolling back file %s: %s", f.Id, err.Error())
			err = nil
		}
		err = r.Db(viper.GetString("DBName")).Table(viper.GetString("FilePieceTable")).Filter(map[string]interface{}{"ParentId": f.Id}).Delete().Exec(session)
		if err != nil {
			log.Printf("problem rolling back file pieces for %s: %s", f.Id, err.Error())
		}
		return
	}
	res.WriteHeader(http.StatusOK)
	res.Write([]byte("," + f.GetUrl(viper.GetString("BaseUrl")) + ","))
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
