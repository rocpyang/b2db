package b2db

import (
"errors"
"reflect"
"strconv"
"strings"
"time"
)

func getTypeName(obj interface{}) (typestr string) {
	typ := reflect.TypeOf(obj)
	typestr = typ.String()

	lastDotIndex := strings.LastIndex(typestr, ".")
	if lastDotIndex != -1 {
		typestr = typestr[lastDotIndex+1:]
	}

	return
}

func snakeCasedName(name string) string {
	newstr := make([]rune, 0)
	firstTime := true

	for _, chr := range name {
		if isUpper := 'A' <= chr && chr <= 'Z'; isUpper {
			if firstTime == true {
				firstTime = false
			} else {
				newstr = append(newstr, '_')
			}
			chr -= ('A' - 'a')
		}
		newstr = append(newstr, chr)
	}

	return string(newstr)
}

func titleCasedName(name string) string {
	newstr := make([]rune, 0)
	upNextChar := true

	for _, chr := range name {
		switch {
		case upNextChar:
			upNextChar = false
			chr -= ('a' - 'A')
		case chr == '_':
			upNextChar = true
			continue
		}

		newstr = append(newstr, chr)
	}

	return string(newstr)
}

func pluralizeString(str string) string {
	if strings.HasSuffix(str, "data") {
		return str
	}
	if strings.HasSuffix(str, "y") {
		str = str[:len(str)-1] + "ie"
	}
	return str + "s"
}
//获取主键所在的键
func getPKColumn(obj interface{})(string,string){
	columnPK:=""
	fieldName:=""
	_,field:=getobjTableName(obj)
	bb := field.Tag
	column:=bb.Get("column")
	if column!="" {
		columnPK=column
		fieldName=field.Name
	}else {
		columnPK=field.Name
		fieldName=field.Name
	}
	return columnPK,fieldName
}
/**
将map的值转给对象
obj	要传值的对象
objMap	储存查询值的map
 */
func scanMapIntoOneToMore(obj interface{}, resultsSlice []map[string][]byte,pKFieldMap map[interface{}]interface{}) error {
	if len(resultsSlice)<=0 {
		return errors.New("查询到零条记录")
	}
	dataStruct := reflect.Indirect(reflect.ValueOf(obj))
	switch dataStruct.Kind() {//判断对象的类型
	case reflect.Slice://为切面
		sliceElementType := dataStruct.Type().Elem()
		fieldNameMap:= pKFieldMap["fieldName"].((map[string]interface{}))
		pK:=pKFieldMap["PK"].(string)
		dataMap:=make(map[string]interface{})
		for _, results := range resultsSlice {
			pkid:=results[pK]
			if dataMap[string(pkid)]==nil{
				oneObj := reflect.New(sliceElementType).Interface()//一方的对象
				oneObjDataStruct := reflect.Indirect(reflect.ValueOf(oneObj))
				scanMapIntoStruct(oneObj, results)
				for key,_:= range fieldNameMap {
					fieldv:=oneObjDataStruct.FieldByName(key)
					oneObjDataStructType := oneObjDataStruct.Type()
					field,_:=oneObjDataStructType.FieldByName(key)
					reflect.New(field.Type.Elem()).Interface()
					//创建切片包含的对象
					moreObj := reflect.New(fieldv.Type().Elem()).Interface()//多方的对象
					err := scanMapIntoStruct(moreObj, results)
					if err != nil {
						return err
					}
					//拼接到多方切片最后
					fieldv.Set(reflect.Append(fieldv, reflect.Indirect(reflect.ValueOf(moreObj))))
				}
				dataMap[string(pkid)]=oneObj
			}else {
				oneObj := dataMap[string(pkid)]//一方的对象
				oneObjDataStruct := reflect.Indirect(reflect.ValueOf(oneObj))
				for key,_:= range fieldNameMap {
					fieldv:=oneObjDataStruct.FieldByName(key)
					oneObjDataStructType := oneObjDataStruct.Type()
					field,_:=oneObjDataStructType.FieldByName(key)
					reflect.New(field.Type.Elem()).Interface()
					//创建切片包含的对象
					moreObj := reflect.New(fieldv.Type().Elem()).Interface()//多方的对象
					err := scanMapIntoStruct(moreObj, results)
					if err != nil {
						return err
					}
					//拼接到多方切片最后
					fieldv.Set(reflect.Append(fieldv, reflect.Indirect(reflect.ValueOf(moreObj))))
				}
			}
		}
		for _,value:= range dataMap {
			dataStruct.Set(reflect.Append(dataStruct, reflect.Indirect(reflect.ValueOf(value))))
		}
		break
	case reflect.Struct://为对象
		fieldNameMap:= pKFieldMap["fieldName"].((map[string]interface{}))
		pK:=pKFieldMap["PK"].(string)
		dataMap:=make(map[string]interface{})
		objsiz:=0
		dataStructType := dataStruct.Type()
		for _, results := range resultsSlice {
			pkid:=results[pK]
			if dataMap[string(pkid)]==nil{
				scanMapIntoStruct(obj, results)
				for key,_:= range fieldNameMap {
					fieldv:=dataStruct.FieldByName(key)
					field,_:=dataStructType.FieldByName(key)
					reflect.New(field.Type.Elem()).Interface()
					//创建切片包含的对象
					newObj := reflect.New(fieldv.Type().Elem()).Interface()
					err := scanMapIntoStruct(newObj, results)
					if err != nil {
						return err
					}
					//拼接到多方切片最后
					fieldv.Set(reflect.Append(fieldv, reflect.Indirect(reflect.ValueOf(newObj))))
				}
				dataMap[string(pkid)]=string(pkid)
				objsiz=objsiz+1
			}else {
				if objsiz>1 {
					return  errors.New("查询出来的对象超过一个")
				}else {
					for key,_:= range fieldNameMap {
						fieldv:=dataStruct.FieldByName(key)
						field,_:=dataStructType.FieldByName(key)
						reflect.New(field.Type.Elem()).Interface()
						//创建切片包含的对象
						newObj := reflect.New(fieldv.Type().Elem()).Interface()
						err := scanMapIntoStruct(newObj, results)
						if err != nil {
							return err
						}
						//拼接到多方切片最后
						fieldv.Set(reflect.Append(fieldv, reflect.Indirect(reflect.ValueOf(newObj))))
					}
				}
			}
		}
		break
	}
	return nil
}
/**
获取一对多一方连接多方的属性
 */
func getOnePKAndMoreFieldName(obj interface{}) (map[interface{}]interface{},error){
	pKFieldMap:= make(map[interface{}]interface{})
	dataStruct := reflect.Indirect(reflect.ValueOf(obj))
	if dataStruct.Kind() != reflect.Struct {
		return pKFieldMap, errors.New("expected a pointer to a struct")
	}
	FieldNameMap:= make(map[string]interface{})
	dataStructType := dataStruct.Type()
	for i := 0; i < dataStructType.NumField(); i++ {
		field := dataStructType.Field(i)
		fieldName := field.Name
		bb := field.Tag
		columnTag := bb.Get("column")//获取表中列的标记
		inlineTag := bb.Get("inline")//获取表中列的标记
		inline := false
		if inlineTag!="" {
			inline = true
		}
		if inline {
			oneToMoreTag := bb.Get("oneToMore")//获取表中一对多的标记
			if oneToMoreTag!="" {
				FieldNameMap[fieldName]=fieldName
			}
		} else {
			beedbTag := bb.Get("beedb")//获取表中列的标记
			if beedbTag=="PK" {
				asTag := bb.Get("as")//获取表字段的别名
				if asTag!="" {
					pKFieldMap["PK"]=asTag
				}else {
					if columnTag!="" {
						pKFieldMap["PK"]=columnTag
					}else {
						pKFieldMap["PK"]=fieldName
					}
				}
			}
		}
	}
	pKFieldMap["fieldName"]=FieldNameMap
	return pKFieldMap,nil
}
/**
将map的值转给对象
obj	要传值的对象
objMap	储存查询值的map
 */
func scanMapIntoStruct(obj interface{}, objMap map[string][]byte) error {
	dataStruct := reflect.Indirect(reflect.ValueOf(obj))
	if dataStruct.Kind() != reflect.Struct {
		return errors.New("expected a pointer to a struct")
	}
	dataStructType := dataStruct.Type()
	for i := 0; i < dataStructType.NumField(); i++ {
		field := dataStructType.Field(i)
		fieldv := dataStruct.Field(i)
		err := scanMapElement(fieldv, field, objMap)
		if err != nil {
			return err
		}
	}

	return nil
}
/**
将map的值映射给对象的属性
 */
func scanMapElement(fieldv reflect.Value, field reflect.StructField, objMap map[string][]byte) error {
	bb := field.Tag
	columnTag := bb.Get("column")//获取表中列的标记
	inlineTag := bb.Get("inline")//获取表中内联的标记
	inline := false
	if inlineTag!="" {
		inline = true
	}
	if inline {
		if field.Type.Kind() == reflect.Struct && field.Type.String() != "time.Time" {
			for i := 0; i < field.Type.NumField(); i++ {
				err := scanMapElement1(fieldv.Field(i), field.Type.Field(i), objMap)
				if err != nil {
					return err
				}
			}
		} else {
			return errors.New("A non struct type can't be inline.")
		}
	}
	//
	asTag := bb.Get("as")//获取表字段的别名
	sqlFieldName:=""//sql中对象的字段名称
	if asTag=="" {
		if columnTag!="" {
			sqlFieldName = columnTag
		}else {
			objFieldName:=field.Name
			sqlFieldName = objFieldName
		}
	}else {
		sqlFieldName=asTag
	}

	// not inline
	data, ok := objMap[sqlFieldName]

	if !ok {
		return nil
	}

	var v interface{}

	switch field.Type.Kind() {

	case reflect.Slice:
		v = data
	case reflect.String:
		v = string(data)
	case reflect.Bool:
		v = string(data) == "1"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32:
		x, err := strconv.Atoi(string(data))
		if err != nil {
			return errors.New("arg " + sqlFieldName + " as int: " + err.Error())
		}
		v = x
	case reflect.Int64:
		x, err := strconv.ParseInt(string(data), 10, 64)
		if err != nil {
			return errors.New("arg " + sqlFieldName + " as int: " + err.Error())
		}
		v = x
	case reflect.Float32, reflect.Float64:
		x, err := strconv.ParseFloat(string(data), 64)
		if err != nil {
			return errors.New("arg " + sqlFieldName + " as float64: " + err.Error())
		}
		v = x
	case reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		x, err := strconv.ParseUint(string(data), 10, 64)
		if err != nil {
			return errors.New("arg " + sqlFieldName + " as int: " + err.Error())
		}
		v = x
		//Supports Time type only (for now)
	case reflect.Struct:
		if fieldv.Type().String() != "time.Time" {
			return errors.New("unsupported struct type in Scan: " + fieldv.Type().String())
		}

		x, err := time.Parse("2006-01-02 15:04:05", string(data))
		if err != nil {
			x, err = time.Parse("2006-01-02 15:04:05.000 -0700", string(data))

			if err != nil {
				return errors.New("unsupported time format: " + string(data))
			}
		}

		v = x
	default:
		return errors.New("unsupported type in Scan: " + reflect.TypeOf(v).String())
	}
	if fieldv.CanSet() {
		fieldv.Set(reflect.ValueOf(v))
		return nil
	}else {
		return errors.New("error: fieldv.CanSet(): false")
	}
}
/**
将map的值映射给对象的属性
 */
func scanMapElement1(fieldv reflect.Value, field reflect.StructField, objMap map[string][]byte) error {
	bb := field.Tag
	columnTag := bb.Get("column")//获取表中列的标记
	inlineTag := bb.Get("inline")//获取表中内联的标记
	inline := false
	if inlineTag!="" {
		inline = true
	}
	if inline {
		return nil
	}
	//
	asTag := bb.Get("as")//获取表字段的别名
	sqlFieldName:=""//sql中对象的字段名称
	if asTag=="" {
		if columnTag!="" {
			sqlFieldName = columnTag
		}else {
			objFieldName:=field.Name
			sqlFieldName = objFieldName
		}
	}else {
		sqlFieldName=asTag
	}

	// not inline
	data, ok := objMap[sqlFieldName]

	if !ok {
		return nil
	}

	var v interface{}

	switch field.Type.Kind() {

	case reflect.Slice:
		v = data
	case reflect.String:
		v = string(data)
	case reflect.Bool:
		v = string(data) == "1"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32:
		x, err := strconv.Atoi(string(data))
		if err != nil {
			return errors.New("arg " + sqlFieldName + " as int: " + err.Error())
		}
		v = x
	case reflect.Int64:
		x, err := strconv.ParseInt(string(data), 10, 64)
		if err != nil {
			return errors.New("arg " + sqlFieldName + " as int: " + err.Error())
		}
		v = x
	case reflect.Float32, reflect.Float64:
		x, err := strconv.ParseFloat(string(data), 64)
		if err != nil {
			return errors.New("arg " + sqlFieldName + " as float64: " + err.Error())
		}
		v = x
	case reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		x, err := strconv.ParseUint(string(data), 10, 64)
		if err != nil {
			return errors.New("arg " + sqlFieldName + " as int: " + err.Error())
		}
		v = x
		//Supports Time type only (for now)
	case reflect.Struct:
		if fieldv.Type().String() != "time.Time" {
			return errors.New("unsupported struct type in Scan: " + fieldv.Type().String())
		}

		x, err := time.Parse("2006-01-02 15:04:05", string(data))
		if err != nil {
			x, err = time.Parse("2006-01-02 15:04:05.000 -0700", string(data))

			if err != nil {
				return errors.New("unsupported time format: " + string(data))
			}
		}

		v = x
	default:
		return errors.New("unsupported type in Scan: " + reflect.TypeOf(v).String())
	}
	if fieldv.CanSet() {
		fieldv.Set(reflect.ValueOf(v))
		return nil
	}else {
		return errors.New("error: fieldv.CanSet(): false")
	}
}
/**
将本身对象转换成map
返回查询的map,参数的map,错误信息
 */
func scanSelfStructIntoMap(obj interface{}) (map[string]interface{}, error) {
	dataStruct := reflect.Indirect(reflect.ValueOf(obj))
	if dataStruct.Kind() != reflect.Struct {
		return nil, errors.New("expected a pointer to a struct")
	}
	tableName,_:=getobjTableName(obj)
	dataStructType := dataStruct.Type()
	selMapp := make(map[string]interface{})
	for i := 0; i < dataStructType.NumField(); i++ {
		field := dataStructType.Field(i)
		fieldName := field.Name
		bb := field.Tag
		columnTag := bb.Get("column")//获取表中列的标记
		asTag := bb.Get("as")//获取表字段的别名
		var mapKey string
		inlineTag := bb.Get("inline")//获取表中内联的标记
		inline := false
		if inlineTag!="" {
			inline = true
		}
		if len(columnTag) > 0 {
			//TODO: support tags that are common in json like omitempty
			if columnTag == "" {
				continue
			}
			//制造出是否有别名的语句
			if asTag!="" {//as标签不为空
				columnTag=tableName+"."+columnTag+" as "+asTag
			}else {
				columnTag=tableName+"."+columnTag
			}
			mapKey = columnTag//查询字段的sql=columnTag+as+astag
		} else {
			mapKey = tableName+"."+fieldName
		}
		if inline {
			continue
		} else {
			value := dataStruct.FieldByName(fieldName).Interface()
			selMapp[mapKey] = value
		}
	}
	return selMapp,nil
}
/**
将对象转换成map
返回查询的map,参数的map,错误信息
 */
func scanSelfColumnIntoMap(obj interface{}) (map[string]interface{}, error) {
	dataStruct := reflect.Indirect(reflect.ValueOf(obj))
	if dataStruct.Kind() != reflect.Struct {
		return nil, errors.New("expected a pointer to a struct")
	}
	dataStructType := dataStruct.Type()
	selMapp := make(map[string]interface{})
	for i := 0; i < dataStructType.NumField(); i++ {
		field := dataStructType.Field(i)
		fieldName := field.Name
		bb := field.Tag
		columnTag := bb.Get("column")//获取表中列的标记
		var mapKey string
		inlineTag := bb.Get("inline")//获取表中内联的标记
		inline := false
		if inlineTag!="" {
			inline = true
		}
		if inline {
			continue
		} else {
			if columnTag=="" {
				mapKey=fieldName
			}else {
				mapKey=columnTag
			}
			value := dataStruct.FieldByName(fieldName).Interface()
			selMapp[mapKey] = value
		}
	}
	return selMapp,nil
}
/**
将对象转换成map
返回查询的map,参数的map,错误信息
 */
func scanStructIntoMap(obj interface{}) (map[string]interface{}, error) {
	dataStruct := reflect.Indirect(reflect.ValueOf(obj))
	if dataStruct.Kind() != reflect.Struct {
		return nil, errors.New("expected a pointer to a struct")
	}
	tableName,_:=getobjTableName(obj)
	dataStructType := dataStruct.Type()
	selMapp := make(map[string]interface{})
	for i := 0; i < dataStructType.NumField(); i++ {
		field := dataStructType.Field(i)
		fieldv := dataStruct.Field(i)
		fieldName := field.Name
		bb := field.Tag
		columnTag := bb.Get("column")//获取表中列的标记
		asTag := bb.Get("as")//获取表字段的别名
		var mapKey string
		inlineTag := bb.Get("inline")//获取表中内联的标记
		inline := false
		if inlineTag!="" {
			inline = true
		}
		if columnTag!="" {
			//TODO: support tags that are common in json like omitempty
			if columnTag == "" {
				continue
			}
			//制造出是否有别名的语句
			if asTag!="" {//as标签不为空
				columnTag=tableName+"."+columnTag+" as "+asTag
			}else {
				columnTag=tableName+"."+columnTag
			}
			mapKey = columnTag//查询字段的sql=columnTag+as+astag
		} else {
			mapKey = tableName+"."+fieldName
		}
		if inline {
			switch field.Type.Kind() {

			case reflect.Slice:
				selMapp2, err2 := scanStructIntoMap1(reflect.New(field.Type.Elem()).Interface())
				if err2 != nil {
					return selMapp2,err2
				}
				for k, v := range selMapp2 {
					selMapp[k] = v
				}
				break
			case reflect.Struct:
				// get an inner map and then put it inside the outer map
				selMapp2, err2 := scanStructIntoMap1(reflect.New(fieldv.Type()).Interface())
				if err2 != nil {
					return selMapp2,err2
				}
				for k, v := range selMapp2 {
					selMapp[k] = v
				}
				break
			default:
				break
			}

		} else {
			value := dataStruct.FieldByName(fieldName).Interface()
			selMapp[mapKey] = value
		}
	}
	return selMapp,nil
}
/**
将对象转换成map
返回查询的map,参数的map,错误信息
 */
func scanStructIntoMap1(obj interface{}) (map[string]interface{}, error) {
	dataStruct := reflect.Indirect(reflect.ValueOf(obj))
	if dataStruct.Kind() != reflect.Struct {
		return nil, errors.New("expected a pointer to a struct")
	}
	tableName,_:=getobjTableName(obj)
	dataStructType := dataStruct.Type()
	selMapp := make(map[string]interface{})
	for i := 0; i < dataStructType.NumField(); i++ {
		field := dataStructType.Field(i)
		fieldName := field.Name
		bb := field.Tag
		columnTag := bb.Get("column")//获取表中列的标记
		asTag := bb.Get("as")//获取表字段的别名
		var mapKey string
		inlineTag := bb.Get("inline")//获取表中内联的标记
		inline := false
		if inlineTag!="" {
			inline = true
		}
		if columnTag!="" {
			//TODO: support tags that are common in json like omitempty
			if columnTag== "" {
				continue
			}
			//制造出是否有别名的语句
			if asTag!="" {//as标签不为空
				columnTag=tableName+"."+columnTag+" as "+asTag
			}else {
				columnTag=tableName+"."+columnTag
			}
			mapKey = columnTag//查询字段的sql=columnTag+as+astag
		} else {
			mapKey = tableName+"."+fieldName
		}
		if inline {
			continue
		} else {
			value := dataStruct.FieldByName(fieldName).Interface()
			selMapp[mapKey] = value
		}
	}
	return selMapp,nil
}
func StructName(s interface{}) string {
	v := reflect.TypeOf(s)
	for v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	return v.Name()
}

/**
获取表格一对一的连接条件
 */
func getOneToOneConnTerm(output interface{})string{
	dataStruct := reflect.Indirect(reflect.ValueOf(output))
	Term := ""
	if dataStruct.Kind() != reflect.Struct {
		return Term
	}
	objTableName1,_:=getobjTableName(output)
	dataStructType := dataStruct.Type()
	for i := 0; i < dataStructType.NumField(); i++ {
		field := dataStructType.Field(i)
		bb := field.Tag
		IdcolumnTag := bb.Get("oneToOne")//获取表oneToOne的标记
		if IdcolumnTag!="" {
			fieldv := dataStruct.Field(i)
			obj2:=reflect.New(fieldv.Type()).Interface()
			objTableName2,field:=getobjTableName(obj2)
			tag := field.Tag
			Term=objTableName1+"."+IdcolumnTag+"="+objTableName2+"."+tag.Get("column")
		}
	}
	return Term
}/**
获取表格一对一的连接条件
 */
func getOneToMoreConnTerm(output interface{})(string,string){
	dataStruct := reflect.Indirect(reflect.ValueOf(output))
	Term := ""
	moreTable :=""
	if dataStruct.Kind() != reflect.Struct {
		return Term,moreTable
	}
	objTableName1,_:=getobjTableName(output)
	dataStructType := dataStruct.Type()
	var pk string
	var objTableName2 string
	var Idcolumn string
	for i := 0; i < dataStructType.NumField(); i++ {
		field := dataStructType.Field(i)
		bb := field.Tag
		beedbTag := bb.Get("beedb")//获取表oneToOne的标记
		if beedbTag!="" {
			columnTag := bb.Get("column")//获取表oneToOne的标记
			pk=columnTag
		}
		IdcolumnTag := bb.Get("oneToMore")//获取表oneToOne的标记
		if IdcolumnTag!="" {
			Idcolumn=IdcolumnTag
			objTableName2=bb.Get("table")
		}
	}
	Term=objTableName1+"."+pk+"="+objTableName2+"."+Idcolumn
	moreTable=objTableName2
	return Term,moreTable
}
/**
获取查询的所有表名称
 */
func getTableName(s interface{}) string {
	dataStruct := reflect.Indirect(reflect.ValueOf(s))
	formTable := ""
	if dataStruct.Kind() != reflect.Struct {
		return formTable
	}
	dataStructType := dataStruct.Type()
	for i := 0; i < dataStructType.NumField(); i++ {
		field := dataStructType.Field(i)
		bb := field.Tag
		tableTag := bb.Get("table")//获取表所在的标记
		if tableTag!="" {
			formTable=formTable+","+tableTag
		}
	}
	return formTable[1:len(formTable)]
}
/**
获取当前对象所在的表和主键所在的列
 */
func  getTableNameAndPKcolumn(output interface{})(string,string){
	var tableName string
	var column string
	if reflect.TypeOf(reflect.Indirect(reflect.ValueOf(output)).Interface()).Kind() == reflect.Slice {
		sliceValue := reflect.Indirect(reflect.ValueOf(output))
		sliceElementType := sliceValue.Type().Elem()
		for i := 0; i < sliceElementType.NumField(); i++ {
			bb := sliceElementType.Field(i).Tag
			columnTag := bb.Get("column")//获取表中列的标记
			if bb.Get("beedb") == "PK" || reflect.ValueOf(bb).String() == "PK" {
				tableNameTag := bb.Get("table")//获取表中表的标记
				tableName=tableNameTag
				if columnTag == "" {
					column=sliceElementType.Field(i).Name
				}else {
					column=columnTag
				}
			}
		}
	} else {
		tt := reflect.TypeOf(reflect.Indirect(reflect.ValueOf(output)).Interface())
		for i := 0; i < tt.NumField(); i++ {
			bb := tt.Field(i).Tag
			columnTag := bb.Get("column")//获取表中列的标记
			if bb.Get("beedb") == "PK" || reflect.ValueOf(bb).String() == "PK" {
				tableNameTag := bb.Get("table")//获取表中表的标记
				tableName=tableNameTag
				if columnTag == "" {
					column=tt.Field(i).Name
				}else {
					column=columnTag
				}
			}
		}
	}
	return tableName,column

}
/**
根据对象获取对象的表名称
 */
func getobjTableName(s interface{}) (string,reflect.StructField){
	dataStruct := reflect.Indirect(reflect.ValueOf(s))
	tableName := ""
	var field reflect.StructField
	if dataStruct.Kind() != reflect.Struct {
		return tableName,field
	}
	dataStructType := dataStruct.Type()
	for i := 0; i < dataStructType.NumField(); i++ {
		field = dataStructType.Field(i)
		bb := field.Tag
		beedbTag:=bb.Get("beedb")
		if beedbTag!= ""&&beedbTag=="PK"  {
			tableName=bb.Get("table")
			break
		} else {
			continue
		}
	}
	return tableName,field
}
/**
查询对象所有表的名称
 */
func scanTableName(s interface{}) string {
	if reflect.TypeOf(reflect.Indirect(reflect.ValueOf(s)).Interface()).Kind() == reflect.Slice {
		sliceValue := reflect.Indirect(reflect.ValueOf(s))
		sliceElementType := sliceValue.Type().Elem()
		for i := 0; i < sliceElementType.NumField(); i++ {
			bb := sliceElementType.Field(i).Tag
			if len(bb.Get("tname")) > 0 {
				return bb.Get("tname")
			}
		}
	} else {
		tt := reflect.TypeOf(reflect.Indirect(reflect.ValueOf(s)).Interface())
		for i := 0; i < tt.NumField(); i++ {
			bb := tt.Field(i).Tag
			if len(bb.Get("tname")) > 0 {
				return bb.Get("tname")
			}
		}
	}
	return ""

}

func stringArrayContains(needle string, haystack []string) bool {
	for _, v := range haystack {
		if needle == v {
			return true
		}
	}
	return false
}
