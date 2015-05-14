package models

import (
	r "github.com/dancannon/gorethink"
	"time"
)

type User struct {
	Id             string `gorethink:"id,omitempty"`
	Email          string
	ApiKey         string
	LastActivity   time.Time
	HashedPassword string
	Active         bool
	Epoch          time.Time
}

type File struct {
	Id           string `gorethink:"id"`
	UserId       string
	FileName     string
	Private      bool
	AccessKey    string
	Epoch        time.Time
	LastAccessed time.Time
	Hits         int
	FileSize     int64
}

type FilePiece struct {
	Seq      int64
	ParentId string
	Data     []byte
}

func (f File) GetUrl(BaseUrl string) string {
	if f.Private {
		return BaseUrl + f.Id + "/" + f.AccessKey
	}
	return BaseUrl + f.Id
}

func (user *User) UpdateLastActivity(session *r.Session, Database string, Table string) error {
	user.LastActivity = time.Now()
	err := r.Db(Database).Table(Table).Update(user).Exec(session)
	if err != nil {
		return err
	}
	return nil
}
