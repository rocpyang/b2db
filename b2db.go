package b2db
import (
"database/sql"
"errors"
"fmt"
"reflect"
"strconv"
"strings"
"time"
"log"
)

var OnDebug = false
//var PluralizeTableNames = false
var beginsession=false//开启事物

type Model struct {
	Tx				*sql.Tx
	Db              *sql.DB
	TableName       string
	LimitStr        int
	OffsetStr       int
	WhereStr        string
	ParamStr        []interface{}
	OrderStr        string
	ColumnStr       string
	PrimaryKey      string
	JoinStr         string
	GroupByStr      string
	HavingStr       string
	QuoteIdentifier string
	ParamIdentifier string
	ParamIteration  int
}

/**
 * Add New sql.DB in the future i will add ConnectionPool.Get()
 */
func New(db *sql.DB, options ...interface{}) (m Model) {
	if len(options) == 0 {
		m = Model{Db: db, ColumnStr: "*", PrimaryKey: "Id", QuoteIdentifier: "`", ParamIdentifier: "?", ParamIteration: 1}
	} else if options[0] == "pg" {
		m = Model{Db: db, ColumnStr: "id", PrimaryKey: "Id", QuoteIdentifier: "\"", ParamIdentifier: options[0].(string), ParamIteration: 1}
	} else if options[0] == "mssql" {
		m = Model{Db: db, ColumnStr: "id", PrimaryKey: "id", QuoteIdentifier: "", ParamIdentifier: options[0].(string), ParamIteration: 1}
	}
	return
}

func (orm *Model) SetTable(tbname string) *Model {
	orm.TableName = tbname
	return orm
}

func (orm *Model) SetPK(pk string) *Model {
	orm.PrimaryKey = pk
	return orm
}

func (orm *Model) Where(querystring interface{}, args ...interface{}) *Model {
	switch querystring := querystring.(type) {
	case string:
		orm.WhereStr = querystring
	case int:
		if orm.ParamIdentifier == "pg" {
			orm.WhereStr = fmt.Sprintf("%v%v%v = $%v", orm.QuoteIdentifier, orm.PrimaryKey, orm.QuoteIdentifier, orm.ParamIteration)
		} else {
			orm.WhereStr = fmt.Sprintf("%v%v%v = ?", orm.QuoteIdentifier, orm.PrimaryKey, orm.QuoteIdentifier)
			orm.ParamIteration++
		}
		args = append(args, querystring)
	}
	orm.ParamStr = args
	return orm
}

func (orm *Model) Limit(start int, size ...int) *Model {
	orm.LimitStr = start
	if len(size) > 0 {
		orm.OffsetStr = size[0]
	}
	return orm
}

func (orm *Model) Offset(offset int) *Model {
	orm.OffsetStr = offset
	return orm
}

func (orm *Model) OrderBy(order string) *Model {
	orm.OrderStr = order
	return orm
}

func (orm *Model) Select(colums string) *Model {
	orm.ColumnStr = colums
	return orm
}

func (orm *Model) ScanPK(output interface{}) *Model {
	if reflect.TypeOf(reflect.Indirect(reflect.ValueOf(output)).Interface()).Kind() == reflect.Slice {
		sliceValue := reflect.Indirect(reflect.ValueOf(output))
		sliceElementType := sliceValue.Type().Elem()
		for i := 0; i < sliceElementType.NumField(); i++ {
			bb := sliceElementType.Field(i).Tag
			if bb.Get("beedb") == "PK" || reflect.ValueOf(bb).String() == "PK" {
				orm.PrimaryKey = sliceElementType.Field(i).Name
			}
		}
	} else {
		tt := reflect.TypeOf(reflect.Indirect(reflect.ValueOf(output)).Interface())
		for i := 0; i < tt.NumField(); i++ {
			bb := tt.Field(i).Tag
			if bb.Get("beedb") == "PK" || reflect.ValueOf(bb).String() == "PK" {
				orm.PrimaryKey = tt.Field(i).Name
			}
		}
	}
	return orm

}
//The join_operator should be one of INNER, LEFT OUTER, CROSS etc - this will be prepended to JOIN
func (orm *Model) Join(join_operator, tablename, condition string) *Model {
	if orm.JoinStr != "" {
		orm.JoinStr = orm.JoinStr + fmt.Sprintf(" %v JOIN %v ON %v", join_operator, tablename, condition)
	} else {
		orm.JoinStr = fmt.Sprintf("%v JOIN %v ON %v", join_operator, tablename, condition)
	}

	return orm
}

func (orm *Model) GroupBy(keys string) *Model {
	orm.GroupByStr = fmt.Sprintf("GROUP BY %v", keys)
	return orm
}

func (orm *Model) Having(conditions string) *Model {
	orm.HavingStr = fmt.Sprintf("HAVING %v", conditions)
	return orm
}

func (orm *Model) FindOne(output interface{}) error {
	orm.ScanPK(output)
	var keys []string
	selMapp, err := scanSelfStructIntoMap(output)
	log.Println("selMapp",selMapp)
	if err != nil {
		return err
	}
	if orm.WhereStr=="" {
		myref:=reflect.ValueOf(output).Elem()
		field := myref.FieldByName(orm.PrimaryKey)
		tableName,column:=getTableNameAndPKcolumn(output)
		orm.Where(tableName+"."+column+"=?",field.Interface())
	}
	if orm.TableName == "" {//获取查询的表名
		orm.TableName,_ = getobjTableName(output)
	}
	// If we've already specific columns with Select(), use that
	if orm.ColumnStr == "*" {//查询的字段
		for key, _ := range selMapp {//如果查询的字符串之前没有定义就用查询的map
			keys = append(keys, key)
		}
		orm.ColumnStr = strings.Join(keys, ", ")
	}
	resultsSlice, err := orm.FindMap()
	if err != nil {
		return err
	}
	if len(resultsSlice) == 0 {
		return errors.New("No record found")
	} else if len(resultsSlice) == 1 {
		results := resultsSlice[0]
		err := scanMapIntoStruct(output, results)
		if err != nil {
			return err
		}
	} else {
		return errors.New("More than one record")
	}
	return nil
}

func (orm *Model) FindAll(rowsSlicePtr interface{}) error {
	orm.ScanPK(rowsSlicePtr)
	sliceValue := reflect.Indirect(reflect.ValueOf(rowsSlicePtr))
	if sliceValue.Kind() != reflect.Slice {
		return errors.New("needs a pointer to a slice")
	}

	sliceElementType := sliceValue.Type().Elem()
	st := reflect.New(sliceElementType)
	var keys []string
	results, err := scanSelfStructIntoMap(st.Interface())
	if err != nil {
		return err
	}
	if orm.TableName == "" {
		orm.TableName,_ = getobjTableName(st.Interface())
	}
	// If we've already specific columns with Select(), use that
	if orm.ColumnStr == "*" {
		for key, _ := range results {
			keys = append(keys, key)
		}
		orm.ColumnStr = strings.Join(keys, ", ")
	}
	resultsSlice, err := orm.FindMap()
	if err != nil {
		return err
	}

	for _, results := range resultsSlice {
		newValue := reflect.New(sliceElementType)
		err := scanMapIntoStruct(newValue.Interface(), results)
		if err != nil {
			return err
		}
		sliceValue.Set(reflect.Append(sliceValue, reflect.Indirect(reflect.ValueOf(newValue.Interface()))))
	}
	return nil
}

func (orm *Model) FindMap() (resultsSlice []map[string][]byte, err error) {
	defer orm.InitModel()
	sqls := orm.generateSql()
	if OnDebug {
		fmt.Println(sqls)
		fmt.Println(orm)
	}
	s, err := orm.Db.Prepare(sqls)
	if err != nil {
		return nil, err
	}
	defer s.Close()
	res, err := s.Query(orm.ParamStr...)
	if err != nil {
		return nil, err
	}
	defer res.Close()
	fields, err := res.Columns()
	if err != nil {
		return nil, err
	}
	for res.Next() {
		result := make(map[string][]byte)
		var scanResultContainers []interface{}
		for i := 0; i < len(fields); i++ {
			var scanResultContainer interface{}
			scanResultContainers = append(scanResultContainers, &scanResultContainer)
		}
		if err := res.Scan(scanResultContainers...); err != nil {
			return nil, err
		}
		for ii, key := range fields {
			rawValue := reflect.Indirect(reflect.ValueOf(scanResultContainers[ii]))
			//if row is null then ignore
			if rawValue.Interface() == nil {
				continue
			}
			aa := reflect.TypeOf(rawValue.Interface())
			vv := reflect.ValueOf(rawValue.Interface())
			var str string
			switch aa.Kind() {
			case reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				str = strconv.FormatInt(vv.Int(), 10)
				result[key] = []byte(str)
			case reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
				str = strconv.FormatUint(vv.Uint(), 10)
				result[key] = []byte(str)
			case reflect.Float32, reflect.Float64:
				str = strconv.FormatFloat(vv.Float(), 'f', -1, 64)
				result[key] = []byte(str)
			case reflect.Slice:
				if aa.Elem().Kind() == reflect.Uint8 {
					result[key] = rawValue.Interface().([]byte)
					break
				}
			case reflect.String:
				str = vv.String()
				result[key] = []byte(str)
				//时间类型
			case reflect.Struct:
				str = rawValue.Interface().(time.Time).Format("2006-01-02 15:04:05.000 -0700")
				result[key] = []byte(str)
			case reflect.Bool:
				if vv.Bool() {
					result[key] = []byte("1")
				} else {
					result[key] = []byte("0")
				}
			}
		}
		resultsSlice = append(resultsSlice, result)
	}
	return resultsSlice, nil
}
func (orm *Model) FindOneToOne(output interface{}) error {
	orm.ScanPK(output)
	var keys []string
	selMapp, err := scanStructIntoMap(output)
	log.Println("selMapp",selMapp)
	if err != nil {
		return err
	}
	if orm.TableName == "" {//获取查询的表名
		orm.TableName = getTableName(output)
	}
	// If we've already specific columns with Select(), use that
	if orm.ColumnStr == "*" {//查询的字段
		for key, _ := range selMapp {//如果查询的字符串之前没有定义就用查询的map
			keys = append(keys, key)
		}
		orm.ColumnStr = strings.Join(keys, ", ")
	}
	if orm.WhereStr=="" {
		myref:=reflect.ValueOf(output).Elem()
		field := myref.FieldByName(orm.PrimaryKey)
		term:=getOneToOneConnTerm(output)
		tableName,column:=getTableNameAndPKcolumn(output)
		orm.Where(term+" AND "+tableName+"."+column+"=?",field.Interface())
	}
	//orm.Limit(1)
	resultsSlice, err := orm.FindMap()
	log.Println("resultsSlice",resultsSlice)
	if err != nil {
		return err
	}
	if len(resultsSlice) == 0 {
		return errors.New("No record found")
	} else if len(resultsSlice) == 1 {
		results := resultsSlice[0]
		err := scanMapIntoStruct(output, results)
		if err != nil {
			return err
		}
	} else {
		return errors.New("More than one record")
	}
	return nil
}
func (orm *Model) FindOneToMore(output interface{}) error {
	orm.ScanPK(output)
	var keys []string
	selMapp, err := scanStructIntoMap(output)
	if err != nil {
		return err
	}
	one_objTable,_:=getobjTableName(output)//获取，一方的表
	if orm.TableName == "" {//获取查询的表名
		orm.TableName = one_objTable
	}
	// If we've already specific columns with Select(), use that
	if orm.ColumnStr == "*" {//查询的字段
		for key, _ := range selMapp {//如果查询的字符串之前没有定义就用查询的map
			keys = append(keys, key)
		}
		orm.ColumnStr = strings.Join(keys, ", ")
	}
	if orm.WhereStr=="" {
		myref:=reflect.ValueOf(output).Elem()
		field := myref.FieldByName(orm.PrimaryKey)
		tableName,column:=getTableNameAndPKcolumn(output)
		orm.Where(tableName+"."+column+"=?",field.Interface())
	}
	if orm.JoinStr=="" {
		term,moreTable:=getOneToMoreConnTerm(output)//获取两表的连接关系
		orm.Join("LEFT",moreTable,term)
	}
	resultsSlice, err := orm.FindMap()
	if err != nil {
		return err
	}
	if len(resultsSlice) == 0 {
		return errors.New("No record found")
	} else  {
		pKFieldMap,er:= getOnePKAndMoreFieldName(output)
		if er!=nil {
			return er
		}
		err := scanMapIntoOneToMore(output, resultsSlice,pKFieldMap)
		if err != nil {
			return err
		}
	}
	return nil
}
func (orm *Model) FindMoreToMore(slice interface{}) error {
	sliceValue := reflect.Indirect(reflect.ValueOf(slice))
	if sliceValue.Kind() != reflect.Slice {
		return errors.New("使用切面来查询")
	}
	obj:=reflect.New(sliceValue.Type().Elem()).Interface()
	orm.ScanPK(obj)
	var keys []string
	selMapp, err := scanStructIntoMap(obj)
	if err != nil {
		return err
	}
	one_objTable,_:=getobjTableName(obj)//获取，一方的表
	if orm.TableName == "" {//获取查询的表名
		orm.TableName = one_objTable
	}
	// If we've already specific columns with Select(), use that
	if orm.ColumnStr == "*" {//查询的字段
		for key, _ := range selMapp {//如果查询的字符串之前没有定义就用查询的map
			keys = append(keys, key)
		}
		orm.ColumnStr = strings.Join(keys, ", ")
	}
	if orm.WhereStr=="" {
		myref:=reflect.ValueOf(obj).Elem()
		field := myref.FieldByName(orm.PrimaryKey)
		tableName,column:=getTableNameAndPKcolumn(obj)
		orm.Where(tableName+"."+column+"=?",field.Interface())
	}
	if orm.JoinStr=="" {
		term,moreTable:=getOneToMoreConnTerm(obj)//获取两表的连接关系
		orm.Join("LEFT",moreTable,term)
	}
	resultsSlice, err := orm.FindMap()
	if err != nil {
		return err
	}
	if len(resultsSlice) == 0 {
		return errors.New("No record found")
	} else  {
		pKFieldMap,er:= getOnePKAndMoreFieldName(obj)
		if er!=nil {
			return er
		}
		err := scanMapIntoOneToMore(slice, resultsSlice,pKFieldMap)
		if err != nil {
			return err
		}
	}
	return nil
}
func (orm *Model) generateSql() (a string) {
	if orm.ParamIdentifier == "mssql" {
		if orm.OffsetStr > 0 {
			a = fmt.Sprintf("select ROW_NUMBER() OVER(order by %v )as rownum,%v from %v",
				orm.PrimaryKey,
				orm.ColumnStr,
				orm.TableName)
			if orm.WhereStr != "" {
				a = fmt.Sprintf("%v WHERE %v", a, orm.WhereStr)
			}
			a = fmt.Sprintf("select * from (%v) "+
				"as a where rownum between %v and %v",
				a,
				orm.OffsetStr,
				orm.LimitStr)
		} else if orm.LimitStr > 0 {
			a = fmt.Sprintf("SELECT top %v %v FROM %v", orm.LimitStr, orm.ColumnStr, orm.TableName)
			if orm.WhereStr != "" {
				a = fmt.Sprintf("%v WHERE %v", a, orm.WhereStr)
			}
			if orm.GroupByStr != "" {
				a = fmt.Sprintf("%v %v", a, orm.GroupByStr)
			}
			if orm.HavingStr != "" {
				a = fmt.Sprintf("%v %v", a, orm.HavingStr)
			}
			if orm.OrderStr != "" {
				a = fmt.Sprintf("%v ORDER BY %v", a, orm.OrderStr)
			}
		} else {
			a = fmt.Sprintf("SELECT %v FROM %v", orm.ColumnStr, orm.TableName)
			if orm.WhereStr != "" {
				a = fmt.Sprintf("%v WHERE %v", a, orm.WhereStr)
			}
			if orm.GroupByStr != "" {
				a = fmt.Sprintf("%v %v", a, orm.GroupByStr)
			}
			if orm.HavingStr != "" {
				a = fmt.Sprintf("%v %v", a, orm.HavingStr)
			}
			if orm.OrderStr != "" {
				a = fmt.Sprintf("%v ORDER BY %v", a, orm.OrderStr)
			}
		}
	} else {
		a = fmt.Sprintf("SELECT %v FROM %v", orm.ColumnStr, orm.TableName)
		if orm.JoinStr != "" {
			a = fmt.Sprintf("%v %v", a, orm.JoinStr)
		}
		if orm.WhereStr != "" {
			a = fmt.Sprintf("%v WHERE %v", a, orm.WhereStr)
		}
		if orm.GroupByStr != "" {
			a = fmt.Sprintf("%v %v", a, orm.GroupByStr)
		}
		if orm.HavingStr != "" {
			a = fmt.Sprintf("%v %v", a, orm.HavingStr)
		}
		if orm.OrderStr != "" {
			a = fmt.Sprintf("%v ORDER BY %v", a, orm.OrderStr)
		}
		if orm.OffsetStr > 0 {
			a = fmt.Sprintf("%v LIMIT %v OFFSET %v", a, orm.LimitStr, orm.OffsetStr)
		} else if orm.LimitStr > 0 {
			a = fmt.Sprintf("%v LIMIT %v", a, orm.LimitStr)
		}
	}
	return
}

//Execute sql
//2017-09-15如果开启事物就使用Tx对数据库操作，
//2017-09-15如果未开始事物使用原方法操作
func (orm *Model) Exec(finalQueryString string, args ...interface{}) (sql.Result, error) {
	var err error
	var rs *sql.Stmt
	if beginsession{
		rs, err =orm.Tx.Prepare(finalQueryString)
	}else {
		rs, err = orm.Db.Prepare(finalQueryString)
	}
	if err != nil {
		return nil, err
	}
	defer rs.Close()
	res, err := rs.Exec(args...)
	if err != nil {
		return nil, err
	}
	return res, nil
}

//if the struct has PrimaryKey == 0 insert else update
func (orm *Model) Save(output interface{}) error {
	//获取主键所在的列
	columnPK,fieldName:=getPKColumn(output)
	orm.PrimaryKey=columnPK
	results, err := scanSelfColumnIntoMap(output)
	log.Println("results",results)
	log.Println("orm",orm)
	if err != nil {
		return err
	}
	if orm.TableName == "" {
		orm.TableName,_= getobjTableName(output)
	}
	id := results[orm.PrimaryKey]
	if id == nil {
		return fmt.Errorf("Unable to save because primary key %q was not found in struct", orm.PrimaryKey)
	}
	switch reflect.ValueOf(id).Type().Kind(){
	case reflect.String:
		idcolumn:=""
		if orm.WhereStr=="" {
			myref:=reflect.ValueOf(output).Elem()
			field := myref.FieldByName(fieldName)
			tableName,column:=getTableNameAndPKcolumn(output)
			idcolumn=column
			orm.Where(tableName+"."+column+"=?",field.Interface())
		}
		paramStr:=orm.ParamStr
		wherestr:=orm.WhereStr
		tableName:=orm.TableName
		if orm.TableName == "" {//获取查询的表名
			orm.TableName=tableName
		}
		// If we've already specific columns with Select(), use that
		var keys []string
		if orm.ColumnStr == "*" {//查询的字段
			for key, _ := range results {//如果查询的字符串之前没有定义就用查询的map
				keys = append(keys, key)
			}
			orm.ColumnStr = strings.Join(keys, ", ")
		}
		resultsSlice, _ := orm.FindMap()
		if len(resultsSlice)<=0 {
			//添加
			if orm.TableName == "" {//获取查询的表名
				orm.TableName=tableName
			}
			if reflect.ValueOf(id).String()!="" {
				_,err := orm.Insert(results)
				if err != nil {
					return err
				}
			}else {
				return fmt.Errorf("columnPK不能为空，存储的columnPK为空",columnPK)
			}
		}else {
			delete(results, idcolumn)
			//修改
			if orm.WhereStr=="" {
				orm.Where(wherestr,paramStr...)
			}
			if orm.TableName == "" {//获取查询的表名
				orm.TableName=tableName
			}
			_, err := orm.Update(results)
			if err != nil {
				return err
			}
		}

		break
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32:
		tableName,column:=getTableNameAndPKcolumn(output)
		delete(results, column)
		if reflect.ValueOf(id).Int() == 0 {//插入
			structPtr := reflect.ValueOf(output)
			structVal := structPtr.Elem()
			structField := structVal.FieldByName(fieldName)
			id, err := orm.Insert(results)
			if err != nil {
				return err
			}
			var v interface{}
			x, err := strconv.Atoi(strconv.FormatInt(id, 10))
			if err != nil {
				return err
			}
			v = x
			structField.Set(reflect.ValueOf(v))
			return nil
		} else {//修改
			myref:=reflect.ValueOf(output).Elem()
			field:= myref.FieldByName(fieldName)
			if orm.WhereStr=="" {
				orm.Where(tableName+"."+column+"=?",field.Interface())
			}
			if orm.TableName == "" {//获取查询的表名
				orm.TableName=tableName
			}
			// If we've already specific columns with Select(), use that
			var keys []string
			if orm.ColumnStr == "*" {//查询的字段
				for key, _ := range results {//如果查询的字符串之前没有定义就用查询的map
					keys = append(keys, key)
				}
				orm.ColumnStr = strings.Join(keys, ", ")
			}
			resultsSlice, _ := orm.FindMap()
			if len(resultsSlice)<=0 {
				return fmt.Errorf("主键为:%d%v",field.Int(),"的行在数据库中不存在，请将主键为0，或者更改主键")
			}else {
				if orm.WhereStr=="" {
					orm.Where(tableName+"."+column+"=?",field.Interface())
				}
				if orm.TableName == "" {//获取查询的表名
					orm.TableName=tableName
				}
				_, err := orm.Update(results)
				if err != nil {
					return err
				}
			}
		}
		break
	default:
		break
	}
	return nil
}

//inert one info
func (orm *Model) Insert(properties map[string]interface{}) (int64, error) {
	defer orm.InitModel()
	var keys []string
	var placeholders []string
	var args []interface{}
	for key, val := range properties {
		keys = append(keys, key)
		if orm.ParamIdentifier == "pg" {
			ds := fmt.Sprintf("$%d", orm.ParamIteration)
			placeholders = append(placeholders, ds)
		} else {
			placeholders = append(placeholders, "?")
		}
		orm.ParamIteration++
		args = append(args, val)
	}
	ss := fmt.Sprintf("%v,%v", orm.QuoteIdentifier, orm.QuoteIdentifier)
	statement := fmt.Sprintf("INSERT INTO %v%v%v (%v%v%v) VALUES (%v)",
		orm.QuoteIdentifier,
		orm.TableName,
		orm.QuoteIdentifier,
		orm.QuoteIdentifier,
		strings.Join(keys, ss),
		orm.QuoteIdentifier,
		strings.Join(placeholders, ", "))
	if OnDebug {
		fmt.Println(statement)
		fmt.Println(orm)
	}
	if orm.ParamIdentifier == "pg" {
		statement = fmt.Sprintf("%v RETURNING %v", statement, snakeCasedName(orm.PrimaryKey))
		var id int64
		if beginsession{
			orm.Tx.QueryRow(statement,args...).Scan(&id)
		}else {
			orm.Db.QueryRow(statement, args...).Scan(&id)
		}
		return id, nil
	} else {
		res, err := orm.Exec(statement, args...)
		if err != nil {
			return -1, err
		}

		id, err := res.LastInsertId()

		if err != nil {
			return -1, err
		}
		return id, nil
	}
	return -1, nil
}

//insert batch info
func (orm *Model) InsertBatch(rows []map[string]interface{}) ([]int64, error) {
	var ids []int64
	tablename := orm.TableName
	if len(rows) <= 0 {
		return ids, nil
	}
	for i := 0; i < len(rows); i++ {
		orm.TableName = tablename
		id, err := orm.Insert(rows[i])
		if err != nil {
			return ids, err
		}

		ids = append(ids, id)
	}
	return ids, nil
}

// update info
func (orm *Model) Update(properties map[string]interface{}) (int64, error) {
	defer orm.InitModel()
	var updates []string
	var args []interface{}
	for key, val := range properties {
		if orm.ParamIdentifier == "pg" {
			ds := fmt.Sprintf("$%d", orm.ParamIteration)
			updates = append(updates, fmt.Sprintf("%v%v%v = %v", orm.QuoteIdentifier, key, orm.QuoteIdentifier, ds))
		} else {
			updates = append(updates, fmt.Sprintf("%v%v%v = ?", orm.QuoteIdentifier, key, orm.QuoteIdentifier))
		}
		args = append(args, val)
		orm.ParamIteration++
	}
	args = append(args, orm.ParamStr...)
	if orm.ParamIdentifier == "pg" {
		if n := len(orm.ParamStr); n > 0 {
			for i := 1; i <= n; i++ {
				orm.WhereStr = strings.Replace(orm.WhereStr, "$"+strconv.Itoa(i), "$"+strconv.Itoa(orm.ParamIteration), 1)
			}
		}
	}
	var condition string
	if orm.WhereStr != "" {
		condition = fmt.Sprintf("WHERE %v", orm.WhereStr)
	} else {
		condition = ""
	}
	statement := fmt.Sprintf("UPDATE %v%v%v SET %v %v",
		orm.QuoteIdentifier,
		orm.TableName,
		orm.QuoteIdentifier,
		strings.Join(updates, ", "),
		condition)
	if OnDebug {
		fmt.Println(statement)
		fmt.Println(orm)
	}
	res, err := orm.Exec(statement, args...)
	if err != nil {
		return -1, err
	}
	id, err := res.RowsAffected()

	if err != nil {
		return -1, err
	}
	return id, nil
}

func (orm *Model) Delete(output interface{}) (int64, error) {
	defer orm.InitModel()
	results, err := scanSelfColumnIntoMap(output)
	if err != nil {
		return 0, err
	}
	tableName,PKcolumn:=getTableNameAndPKcolumn(output)
	if orm.TableName == "" {
		orm.TableName = tableName
	}
	id := results[PKcolumn]
	condition := fmt.Sprintf("%v%v%v='%v'", orm.QuoteIdentifier, PKcolumn, orm.QuoteIdentifier, id)
	statement := fmt.Sprintf("DELETE FROM %v%v%v WHERE %v",
		orm.QuoteIdentifier,
		orm.TableName,
		orm.QuoteIdentifier,
		condition)
	if OnDebug {
		fmt.Println(statement)
		fmt.Println(orm)
	}
	res, err := orm.Exec(statement)
	if err != nil {
		return -1, err
	}
	Affectid, err := res.RowsAffected()

	if err != nil {
		return -1, err
	}
	return Affectid, nil
}

func (orm *Model) DeleteAll(rowsSlicePtr interface{}) (int64, error) {
	defer orm.InitModel()
	tableName,PKcolumn:=getTableNameAndPKcolumn(rowsSlicePtr)
	if orm.TableName == "" {
		//TODO: fix table name
		orm.TableName = tableName
	}
	var ids []string
	val := reflect.Indirect(reflect.ValueOf(rowsSlicePtr))
	if val.Len() == 0 {
		return 0, nil
	}
	for i := 0; i < val.Len(); i++ {
		results, err := scanSelfColumnIntoMap(val.Index(i).Interface())
		if err != nil {
			return 0, err
		}
		id := results[PKcolumn]
		switch id.(type) {
		case string:
			ids = append(ids, id.(string))
		case int, int64, int32:
			str := strconv.Itoa(id.(int))
			ids = append(ids, str)
		}
	}
	condition := fmt.Sprintf("%v%v%v in ('%v')", orm.QuoteIdentifier, PKcolumn, orm.QuoteIdentifier, strings.Join(ids, "','"))
	statement := fmt.Sprintf("DELETE FROM %v%v%v WHERE %v",
		orm.QuoteIdentifier,
		orm.TableName,
		orm.QuoteIdentifier,
		condition)
	if OnDebug {
		fmt.Println(statement)
		fmt.Println(orm)
	}
	res, err := orm.Exec(statement)
	if err != nil {
		return -1, err
	}
	Affectid, err := res.RowsAffected()

	if err != nil {
		return -1, err
	}
	return Affectid, nil
}

func (orm *Model) DeleteRow() (int64, error) {
	defer orm.InitModel()
	var condition string
	if orm.WhereStr != "" {
		condition = fmt.Sprintf("WHERE %v", orm.WhereStr)
	} else {
		condition = ""
	}
	statement := fmt.Sprintf("DELETE FROM %v%v%v %v",
		orm.QuoteIdentifier,
		orm.TableName,
		orm.QuoteIdentifier,
		condition)
	if OnDebug {
		fmt.Println(statement)
		fmt.Println(orm)
	}
	res, err := orm.Exec(statement, orm.ParamStr...)
	if err != nil {
		return -1, err
	}
	Affectid, err := res.RowsAffected()

	if err != nil {
		return -1, err
	}
	return Affectid, nil
}

/**
开启事物
 */
func (orm *Model)Begin()  error {
	var err=error(nil)
	orm.Tx,err=orm.Db.Begin()
	if err!=nil {
		return err
	}
	beginsession=true
	return  err
}
/*
提交数据
 */
func (orm *Model)Commit() error {
	err:=orm.Tx.Commit()
	if err!=nil {
		return err
	}
	beginsession=false
	return nil
}
/**
回滚数据
 */
func (orm *Model)Rollback() error {
	err:=orm.Tx.Rollback()
	if err!=nil {
		return err
	}
	beginsession=false
	return nil
}
/**
初始化对象
 */
func (orm *Model) InitModel() {
	orm.TableName = ""
	orm.LimitStr = 0
	orm.OffsetStr = 0
	orm.WhereStr = ""
	orm.ParamStr = make([]interface{}, 0)
	orm.OrderStr = ""
	orm.ColumnStr = "*"
	orm.PrimaryKey = "id"
	orm.JoinStr = ""
	orm.GroupByStr = ""
	orm.HavingStr = ""
	orm.ParamIteration = 1
}

