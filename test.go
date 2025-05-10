package main

import (
	"fmt"
	"reflect"
)

type User struct {
	Name string `json:"nnn" Index:"U"`
	Age  int
}

func main() {
	user := &User{"John Doe The Fourth", 20}

	field, ok := reflect.TypeOf(user).Elem().FieldByName("Name")
	if !ok {
		panic("Field not found")
	}
	t := reflect.TypeOf(user).Elem()
	for i := range t.NumField() {
		t.Field(i).Tag.Get("Index")
		fmt.Printf("%+v\n", t.Field(i))
		fmt.Printf("%+v\n", t.Field(i).Type.String())
	}
	fmt.Println(field.Tag.Get("Index"))
	// fmt.Println(getStructTag(field))
}

func getStructTag(f reflect.StructField) string {
	return string(f.Tag)
}
