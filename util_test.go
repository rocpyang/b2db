package b2db

import (
	"testing"
	"reflect"
	"errors"
)

type User struct {
	Id int `beedb:"PK"  table:"class" column:"Id" as:"CId"`
	Name string `column:"name" as:"Cname"`
	Students []SQLModel `column:"student" inline:"true" oneToMore:"classId" table:"student"`
}

type SQLModel struct {
	SId int `beedb:"PK" table:"student" column:"Id" as:"SId"`
	user User `table:"class"  inline:"true" oneToOne:"classId"`
	Name string `column:"name"`
	PassWord string `column:"password"`
	ClassId string	`column:"classId"`
}

func TestMapToStruct(t *testing.T) {
	target := &User{}
	input := map[string][]byte{
		"name":     []byte("Test User"),
		"auth":     []byte("1"),
		"id":       []byte("1"),
		"created":  []byte("2014-01-01 10:10:10"),
		"modified": []byte("2014-01-01 10:10:10"),
	}
	err := scanMapIntoStruct(target, input)
	if err != nil {
		t.Errorf(err.Error())
	}

	_, err = scanStructIntoMap(target)

	if err != nil {
		t.Errorf(err.Error())
	}
}

