package rethinkdbutils

import (
	"errors"
	r "gopkg.in/dancannon/gorethink.v1"
	"log"
)

func MakeDbIfNotExist(DbName string, session *r.Session) error {
	log.Printf("Checking if DB %s exists and creating it if it does not.", DbName)
	cur, err := r.DBList().Contains(DbName).Run(session)
	if err != nil {
		log.Println(err.Error())
		return err
	}
	var k interface{}
	err = cur.One(&k)
	if err != nil {
		log.Println(err.Error())
		return err
	}
	check, ok := k.(bool)
	if !ok {
		err = errors.New("Could not turn contains into bool")
		log.Println(err.Error())
		log.Println(k)
		return err
	}
	if !check {
		log.Printf("DB %s does not exist. Creating it now", DbName)
		err := r.DBCreate(DbName).Exec(session)
		if err != nil {
			log.Println(err.Error())
			return err
		}
		return nil
	}
	log.Printf("DB %s already exists", DbName)
	return nil
}

func MakeTableIfNotExist(DbName string, TableName string, session *r.Session) error {
	log.Printf("Checking if  %s.%s exists and creating it if it does not.", DbName, TableName)
	cur, err := r.DB(DbName).TableList().Contains(TableName).Run(session)
	if err != nil {
		log.Println(err.Error())
		return err
	}
	var k interface{}
	err = cur.One(&k)
	if err != nil {
		log.Println(err.Error())
		return err
	}
	check, ok := k.(bool)
	if !ok {
		err = errors.New("Could not turn contains into bool")
		log.Println(err.Error())
		log.Println(k)
		return err
	}
	if !check {
		log.Printf("%s.%s does not exist. Creating it now", DbName, TableName)
		err := r.DB(DbName).TableCreate(TableName).Exec(session)
		if err != nil {
			log.Println(err.Error())
			return err
		}
		return nil

	}
	log.Printf("%s.%s already exists", DbName, TableName)
	return nil
}

func MakeIndexIfNotExist(DbName string, TableName string, IndexName string, session *r.Session) error {
	log.Printf("Checking if  index %s.%s.%s exists and creating it if it does not.", DbName, TableName, IndexName)
	cur, err := r.DB(DbName).Table(TableName).IndexList().Contains(IndexName).Run(session)
	if err != nil {
		log.Println(err.Error())
		return err
	}
	var k interface{}
	err = cur.One(&k)
	if err != nil {
		log.Println(err.Error())
		return err
	}
	check, ok := k.(bool)
	if !ok {
		err = errors.New("Could not turn contains into bool")
		log.Println(err.Error())
		log.Println(k)
		return err
	}
	if !check {
		log.Printf("index %s.%s.%s does not exist. Creating it now", DbName, TableName, IndexName)
		err := r.DB(DbName).Table(TableName).IndexCreate(IndexName).Exec(session)
		if err != nil {
			log.Println(err.Error())
			return err
		}
		return nil
	}
	log.Printf("index %s.%s.%s already exists", DbName, TableName, IndexName)
	return nil
}
