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

type Config struct {
	RethinkDbConnectionString string
	Db                        string
	FileTable                 string
	FilePieceTable            string
	MetaTable                 string
	BaseUrl                   string
	UserTable                 string
}

func (user *User) UpdateLastActivity(session *r.Session, Database string, Table string) error {
	user.LastActivity = time.Now()
	err := r.Db(Database).Table(Table).Update(user).Exec(session)
	if err != nil {
		return err
	}
	return nil
}
