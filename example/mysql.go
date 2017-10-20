package main

import (
	"database/sql"
	"fmt"
	_ "github.com/ziutek/mymysql/godrv"
	"strconv"
	"time"
	"log"
	"b2db"
)


const (
	Username="root"//用户名
	PassWord="root"//密码
	dbname="school"//数据库
)
/**
获取DB连接
 */
func InItOrm() (b2DB b2db.Model) {
	db, err := sql.Open("mymysql", dbname+"/"+Username+"/"+PassWord)
	if err!=nil {
		log.Println(err)
		log.Println("db初始化为空")
	}else {
		b2DB = b2db.New(db)
	}
	return
}

/**
学生表对象
 */
/*
CREATE TABLE `student` (
  `Id` int(11) NOT NULL AUTO_INCREMENT,
  `name` varchar(50) DEFAULT NULL,
  `password` varchar(50) DEFAULT NULL,
  `classId` int(11) DEFAULT NULL,
  PRIMARY KEY (`Id`),
  KEY `classId` (`classId`),
  CONSTRAINT `student_ibfk_1` FOREIGN KEY (`classId`) REFERENCES `class` (`Id`)
) ENGINE=InnoDB AUTO_INCREMENT=57 DEFAULT CHARSET=utf8;
 */
type Student struct{
	SId int `b2db:"PK" table:"student" column:"Id" as:"SId"`
	Class Class `table:"class"  inline:"true" oneToOne:"classId"`
	Name string `column:"name"`
	PassWord string `column:"password"`
	ClassId string	`column:"classId"`
}
/**
班级表对象
 */
/*
CREATE TABLE `class` (
  `Id` int(11) NOT NULL AUTO_INCREMENT,
  `name` varchar(255) DEFAULT NULL,
  PRIMARY KEY (`Id`)
) ENGINE=InnoDB AUTO_INCREMENT=3 DEFAULT CHARSET=utf8 COMMENT='班级';
 */
type Class struct {
	Id int `b2db:"PK"  table:"class" column:"Id" as:"CId"`
	Name string `column:"name" as:"Cname"`
	Students []Student `column:"student" inline:"true" oneToMore:"classId" table:"student"`
}
/**
CREATE TABLE `userinfo` (
	`uid` INT(10) NULL AUTO_INCREMENT,
	`username` VARCHAR(64) NULL,
	`departname` VARCHAR(64) NULL,
	`created` DATE NULL,
	PRIMARY KEY (`uid`)
);
 */
type Userinfo struct {
	Id string `b2db:"PK"  table:"userdeatail" column:"uid" as:"RId"`
	Intro string `column:"intro" as:"Rintro"`
}
func main() {
	orm:= InItOrm()
	b2db.OnDebug = true




	/**
	insert
	 */
	//insert(orm)
	//insertsql(orm)
	//insertbatch(orm)









	/**
	delete
	 */
	//delete(orm)
	//deletesql(orm)
	//deleteall(orm)







	/**
	update
	 */
	//update(orm)
	//updatesql(orm)







	/**
	select
	 */
	stude:=selectone(orm)
	log.Println(stude)
	//student:=selectall(orm)
	//log.Println(student)
	//findmap(orm)
	//findOneToOne:=findOneToOne(orm)
	//log.Println(findOneToOne)
	//class:=FindOneToMore(orm)
	//log.Println(class)
	//class:=FindMoreToMore(orm)
	//log.Println(class[0].Name,class[1].Name)
	//log.Println(len(class[1].Students))






	/**
	groupby
	 */
	//groupby(orm)






	/**
	jointable
	 */
	//jointable(orm)
}
func insert(orm b2db.Model) {
	var student Student
	//student.SId=42
	student.Name = time.Now().String()[0:10]
	student.PassWord = "Test Add Departname"
	student.ClassId = "1"
	err:=orm.Save(&student)
	log.Println(err)
	log.Println(student)
}
func insertsql(orm b2db.Model) {
	add := make(map[string]interface{})
	add["name"] = "cloud develop"
	add["password"] = "2012-12-02"
	add["classId"] = "2"
	in,err:=orm.SetTable("student").Insert(add)
	log.Println(in)
	log.Println(err)
}
func insertbatch(orm b2db.Model) {
	rows := make([]map[string]interface{}, 5)
	for i := 0; i < 5; i++ {
		add := make(map[string]interface{})
		name := "person" + strconv.Itoa(i)
		add["username"] = name
		add["departname"] = "IT"
		add["created"] = time.Now().String()[0:10]
		rows[i] = add
	}
	fmt.Println(rows)
	in,err:=orm.SetTable("userinfo").InsertBatch(rows)
	log.Println(in)
	log.Println(err)
}
func delete(orm b2db.Model) {
	// // //delete one data
	saveone := selectone(orm)
	log.Println(saveone)
	in,err:=orm.Delete(&saveone)
	log.Println(in)
	log.Println(err)
}
func deletesql(orm b2db.Model) {
	//original SQL delete
	orm.SetTable("userinfo").Where("uid=?", 30).DeleteRow()
}
func deleteall(orm b2db.Model) {
	// //delete all data
	allstu := selectall(orm)
	log.Println(allstu)
	in,err:=orm.DeleteAll(&allstu)
	log.Println(in)
	log.Println(err)
}
func update(orm b2db.Model) {
	var student Student
	student.SId=56
	student.Name = time.Now().String()[0:10]
	student.PassWord = "123456"
	student.ClassId = "2"
	err:=orm.Save(&student)
	log.Println(err)
	log.Println(student)
}
func updatesql(orm b2db.Model) {
	t := make(map[string]interface{})
	t["username"] = "yangp"
	in,_:=orm.SetTable("userinfo").SetPK("uid").Where(2).Update(t)
	log.Println(in)
}
func selectone(orm b2db.Model) Student {
	var student Student
	student.SId=55
	err:=orm.FindOne(&student)
	log.Println(err)
	return student
}
func selectall(orm b2db.Model) []Student {
	var allStudent []Student
	orm.Limit(2).Where("Id>30", ).FindAll(&allStudent)
	return allStudent
}
func findmap(orm b2db.Model) {
	//Original SQL Backinfo resultsSlice []map[string][]byte
	//default PrimaryKey id
	c, _ := orm.
	SetTable("student").
		SetPK("Id").
		Where("Id > ?", "10").
		Select("student.Id as SId, student.name, student.password, student.classId").
		FindMap()
	fmt.Println(c)
}
func findOneToOne(orm b2db.Model) Student {
	var student Student
	student.SId=36
	////orm.SetTable("class,student").Where(" student.Id=? AND class.Id=student.classId",41).Find(student)
	orm.FindOneToOne(&student)
	//orm.SetTable("student").Join("LEFT","class","class.Id=student.classId").Where("class.Id=?",1).FindMap()
	return student
}
func FindOneToMore(orm b2db.Model) Class {
	var class Class
	class.Id=1
	orm.FindOneToMore(&class)
	log.Println(len(class.Students))
	return class
}
func FindMoreToMore(orm b2db.Model) []Class {
	var class []Class
	////orm.SetTable("class,student").Where(" student.Id=? AND class.Id=student.classId",41).Find(student)
	orm.Where("class.Id>0").FindMoreToMore(&class)
	//orm.SetTable("student").Join("LEFT","class","class.Id=student.classId").Where("class.Id=?",1).FindMap()
	return class
}
func groupby(orm b2db.Model) {
	//Original SQL Group By
	b, _ := orm.SetTable("student").GroupBy("name").Having("name='123'").FindMap()
	fmt.Println(b)
}
func jointable(orm b2db.Model) {
	//Original SQL Join Table
	a, _ := orm.SetTable("userinfo").Join("LEFT", "userdeatail", "userinfo.uid=userdeatail.uid").Where("userinfo.uid=?", 10).Select("userinfo.uid,userinfo.username,userdeatail.profile").FindMap()
	fmt.Println(a)
}