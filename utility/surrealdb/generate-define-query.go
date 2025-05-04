package surrealdb

import (
	"fmt"
	"reflect"
)

func GenerateDefineQueryWithIndexAndByStruct[TAny any](tableName string, dataStruct TAny, createSchemafull bool) string {
	structEle := reflect.TypeOf(dataStruct)
	DefineString := fmt.Sprintf("DEFINE TABLE %s", tableName)
	SchemaFull := ""
	if createSchemafull {
		SchemaFull = "SCHEMAFULL"
	}
	DefineString += fmt.Sprintf(" %s; \n", SchemaFull)
	for i := range structEle.NumField() {
		// name:=
		field := structEle.Field(i)
		filedName := field.Name
		if jsonName := field.Tag.Get("json"); jsonName != "" {
			filedName = jsonName
		}
		fieldType := field.Type.String()
		if fieldTypeFromTag := field.Tag.Get("fieldType"); fieldTypeFromTag != "" {
			fieldType = fieldTypeFromTag
		}
		DefineString += fmt.Sprintf("DEFINE FIELD %s ON TABLE %s TYPE %s ", filedName, tableName, fieldType)
		if defaultValue := field.Tag.Get("defaultValue"); defaultValue != "" {
			always := ""
			if field.Tag.Get("defaultValueAlways") != "" {
				always = "ALWAYS"
			}
			DefineString += fmt.Sprintf("DEFAULT %s %s ", always, defaultValue)
		}
		DefineString += "; \n"
		if index := field.Tag.Get("Index"); index != "" {
			indexType := ""
			if index == "U" {
				indexType = "UNIQUE"
			}
			DefineString += fmt.Sprintf("DEFINE INDEX %sIndex ON TABLE %s COLUMNS %s %s; \n", filedName, tableName, filedName, indexType)
		}
	}
	return DefineString
}
