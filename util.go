package b2db

import (
	"errors"
	"reflect"
	"strconv"
	"time"
)
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
///**
//将对象转换成map
//返回查询的map,参数的map,错误信息
// */
//func scanSelfColumnIntoMap1(obj interface{}) (map[string]interface{}, error) {
//	dataStruct := reflect.Indirect(reflect.ValueOf(obj))
//	if dataStruct.Kind() != reflect.Struct {
//		return nil, errors.New("expected a pointer to a struct")
//	}
//	dataStructType := dataStruct.Type()
//	selMapp := make(map[string]interface{})
//	for i := 0; i < dataStructType.NumField(); i++ {
//		field := dataStructType.Field(i)
//		fieldName := field.Name
//		bb := field.Tag
//		columnTag := bb.Get("column")//获取表中列的标记
//		var mapKey string
//		inlineTag := bb.Get("inline")//获取表中内联的标记
//		inline := false
//		if inlineTag!="" {
//			inline = true
//		}
//		if inline {
//			continue
//		} else {
//			if columnTag=="" {
//				mapKey=fieldName
//			}else {
//				mapKey=columnTag
//			}
//			value := dataStruct.FieldByName(fieldName).Interface()
//			selMapp[mapKey] = value
//		}
//	}
//	return selMapp,nil
//}
type selectStruct struct {
	tableName string
	oneToOneTable string
	pkFieldName string
	pkcolumn string
	pkPram interface{}
	OneToOneConnTerm string
	OneToMoreConnTerm string
	moreTable string
	pKFieldMap  map[interface{}]interface{}
	fieldpram string
	selOneMap map[string]interface{}
	selAllMap map[string]interface{}
}

func scanStruct(obj interface{}) ( *selectStruct, error) {
	var selStruct selectStruct
	dataStruct := reflect.Indirect(reflect.ValueOf(obj))
	if dataStruct.Kind() != reflect.Struct {
		return nil, errors.New("expected a pointer to a struct")
	}
	tableName,_:=getTableName(obj)
	formTable := ""
	oneToMoreIdColumn := ""
	oneToMoreTableName := ""
	oneToOneTableName := ""
	pKFieldMap:= make(map[interface{}]interface{})
	FieldNameMap:= make(map[string]interface{})
	dataStructType := dataStruct.Type()
	selAllMap := make(map[string]interface{})
	selOneMap := make(map[string]interface{})
	for i := 0; i < dataStructType.NumField(); i++ {
		//dataStructType.
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
		tableTag := bb.Get("table")//获取表所在的标记
		if tableTag!="" {
			formTable=formTable+","+tableTag
		}
		pkTag := bb.Get("b2db")//获取表所在的标记
		if pkTag!="" {
			selStruct.pkFieldName=fieldName
			selStruct.pkcolumn=columnTag
			myref:=reflect.ValueOf(obj).Elem()
			fiel := myref.FieldByName(fieldName)
			selStruct.pkPram=fiel.Interface()
		}
		oneToOneTag := bb.Get("oneToOne")//获取表oneToOne的标记
		if oneToOneTag!="" {
			fieldv := dataStruct.Field(i)
			obj2:=reflect.New(fieldv.Type()).Interface()
			var fiel reflect.StructField
			oneToOneTableName,fiel=getTableName(obj2)
			tag := fiel.Tag
			selStruct.OneToOneConnTerm=tableName+"."+oneToOneTag+"="+oneToOneTableName+"."+tag.Get("column")
		}
		oneToMoreTag := bb.Get("oneToMore")//获取表oneToOne的标记
		if oneToMoreTag!="" {
			oneToMoreIdColumn=oneToMoreTag
			oneToMoreTableName=bb.Get("table")
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
			oneToMoreTag := bb.Get("oneToMore")//获取表中一对多的标记
			if oneToMoreTag!="" {
				FieldNameMap[fieldName]=fieldName
			}
		} else {
			b2dbTag := bb.Get("b2db")//获取表中列的标记
			if b2dbTag=="PK" {
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
		if inline {
			fieldv := dataStruct.Field(i)
			switch field.Type.Kind() {

			case reflect.Slice:
				selMapp2, err2 := scanStructIntoMap1(reflect.New(field.Type.Elem()).Interface())
				if err2 != nil {
					return nil,err2
				}
				for k, v := range selMapp2 {
					selAllMap[k] = v
				}
				break
			case reflect.Struct:
				// get an inner map and then put it inside the outer map
				selMapp2, err2 := scanStructIntoMap1(reflect.New(fieldv.Type()).Interface())
				if err2 != nil {
					return nil,err2
				}
				for k, v := range selMapp2 {
					selAllMap[k] = v
				}
				break
			default:
				break
			}
		} else {
			value := dataStruct.FieldByName(fieldName).Interface()
			selAllMap[mapKey] = value
			selOneMap[mapKey] = value
		}
	}
	selStruct.selAllMap=selAllMap
	selStruct.selOneMap=selOneMap
	selStruct.tableName=tableName
	selStruct.oneToOneTable=tableName+","+oneToOneTableName
	selStruct.OneToMoreConnTerm= tableName+"."+selStruct.pkcolumn+"="+oneToMoreTableName+"."+oneToMoreIdColumn
	selStruct.moreTable=oneToMoreTableName
	pKFieldMap["fieldName"]=FieldNameMap
	selStruct.pKFieldMap=pKFieldMap
	return &selStruct,nil
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
	tableName,_:=getTableName(obj)
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
/**
根据对象获取对象的表名称
 */
func getTableName(s interface{}) (string,reflect.StructField){
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
		b2dbTag:=bb.Get("b2db")
		if b2dbTag!= ""&&b2dbTag=="PK"  {
			tableName=bb.Get("table")
			break
		} else {
			continue
		}
	}
	return tableName,field
}
/**
将对象转换成map
返回查询的map,参数的map,错误信息
 */
func scanSelfColumn(obj interface{})  ( map[string]interface{}, error)  {
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
			if bb.Get("b2db") == "PK" || reflect.ValueOf(bb).String() == "PK" {
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
			if bb.Get("b2db") == "PK" || reflect.ValueOf(bb).String() == "PK" {
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