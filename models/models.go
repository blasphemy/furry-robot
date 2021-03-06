package models

import (
	r "gopkg.in/dancannon/gorethink.v1"
	"strings"
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
	MimeType     string
}

type FilePiece struct {
	Seq      int64
	ParentId string
	Data     []byte
}

func (f File) GetUrl(BaseUrl string) string {
	if !strings.HasSuffix(BaseUrl, "/") {
		BaseUrl = BaseUrl + "/"
	}
	if f.Private {
		return BaseUrl + f.Id + "/" + f.AccessKey
	}
	return BaseUrl + f.Id
}

func (user *User) UpdateLastActivity(session *r.Session, Table string) error {
	user.LastActivity = time.Now()
	err := r.Table(Table).Update(user).Exec(session)
	if err != nil {
		return err
	}
	return nil
}

func GetUserByApiKey(ApiKey string, session *r.Session, Tablename string, debug bool) (User, error) {
	var result User
	if debug {
		result.Active = true
		return result, nil
	}
	cur, err := r.Table(Tablename).Filter(map[string]interface{}{"ApiKey": ApiKey}).Run(session)
	if err != nil {
		return result, err
	}
	err = cur.One(&result)
	if err != nil {
		return result, err
	}
	return result, nil
}
