
*　　* b2db是一个dao层的框架，现在网络上有好多golang Dao的框架，比如beegoo.rm、beedb;在使用过程中感觉beegoo.rm太大了，而beedb不支持事物以及一对一，多对多查询。刚开始的的时候无聊更改了beedb的代码，增接了事物这个功能，后来有时间了就重新写了b2db。下面来说b2db的使用吧：
<!--more-->

*　　* 首先贴上[下载地址](https://github.com/JeonYang/b2db.git)

*　　* 或者：go get https://github.com/JeonYang/b2db.git

## 1. 创建DB连接

* 这个以前有总结可以看一下

## 2. 创建b2DB对象

* b2DB = b2db.New(db)

## 做完上面两步就可以使用b2DB了

## 增加

*　　* 按照对象添加

	var student Student
	//student.SId=42
	student.Name = time.Now().String()[0:10]
	student.PassWord = "Test Add Departname"
	student.ClassId = "1"
	err:=orm.Save(&student)

*　　* 按照map添加

	add := make(map[string]interface{})
	add["name"] = "cloud develop"
	add["password"] = "2012-12-02"
	add["classId"] = "2"
	in,err:=orm.SetTable("student").Insert(add)

*　　* 多个添加

	rows := make([]map[string]interface{}, 5)
	for i := 0; i < 5; i++ {
		add := make(map[string]interface{})
		name := "person" + strconv.Itoa(i)
		add["username"] = name
		add["departname"] = "IT"
		add["created"] = time.Now().String()[0:10]
		rows[i] = add
	}
	in,err:=orm.SetTable("userinfo").InsertBatch(rows)

## 删除

*　　* 删除单个对象

	saveone := selectone(orm)
	log.Println(saveone)
	in,err:=orm.Delete(&saveone)

*　　* 按照一定条件删除

	orm.SetTable("userinfo").Where("uid=?", 30).DeleteRow()

*　　* 同时删除多个对象

	var allStudent []Student
	in,err:=orm.DeleteAll(&allStudent)

## 修改

*　　* 按照对象修改

	var student Student
	//student.SId=42
	student.Name = time.Now().String()[0:10]
	student.PassWord = "Test Add Departname"
	student.ClassId = "1"
	err:=orm.Save(&student)

*　　* 按照一定条件删除

	t := make(map[string]interface{})
	t["username"] = "yangp"
	in,_:=orm.SetTable("userinfo").SetPK("uid").Where(2).Update(t)

## 查询

*　　* 按照对象查找

	var student Student
	student.SId=55
	err:=orm.FindOne(&student)

*　　* 一次查找多个

	var allStudent []Student
	orm.Limit(2).Where("Id>30", ).FindAll(&allStudent)

*　　* 按照一定条件查找

	SetTable("student").
	SetPK("Id").
	Where("Id > ?", "10").
	Select("student.Id as SId, student.name, student.password, student.classId").
	FindMap()

*　　* 一对一查找

	var student Student
	student.SId=36
	orm.FindOneToOne(&student)

*　　* 一对多查找

	var class Class
	class.Id=1
	orm.FindOneToMore(&class)

*　　* 多对多查找

	var class []Class
	orm.Where("class.Id>0").FindMoreToMore(&class)

*　　* GroupBy查找

	b, _ := orm.SetTable("student").GroupBy("name").Having("name='123'").FindMap()

*　　* Join查找

	a, _ := orm.SetTable("userinfo").Join("LEFT", "userdeatail", "userinfo.uid=userdeatail.uid").Where("userinfo.uid=?", 10).Select("userinfo.uid,userinfo.username,userdeatail.profile").FindMap()

## 事物

*　　* 开启事物

	orm.Begin()

*　　* 提交事物

	orm.Commit()