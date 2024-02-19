package main

import (
	"encoding/json"
	"github.com/go-webs/dbgo"
	_ "github.com/go-webs/dbgo-driver-mysql"
	"log"
	"strconv"
	"time"
)

type User struct {
	Id    int64  `db:"id,pk" json:"id"`
	Name  string `db:"name" json:"name"`
	Email string `db:"email" json:"email"`

	TableName string `db:"users" json:"-"`
}

var dg = dbgo.Open("mysql", "root:123456@tcp(localhost:3306)/test?charset=utf8mb4&parseTime=true")

func db() dbgo.Database {
	return dg.NewDatabase()
}
func checkError(err error) {
	if err != nil {
		panic(err.Error())
	}
}
func checkAffectedRows(aff int64) {
	if aff == 0 {
		panic("影响行数为0")
	}
}
func checkLastInsertId(lastId int64) {
	if lastId == 0 {
		panic("插入失败")
	}
}
func main() {
	start := time.Now()
	//to()
	insertGetId()
	//update()
	//delete()
	log.Println(time.Since(start))
}
func to() {
	var user User
	err := db().OrderByDesc("id").To(&user)
	checkError(err)
	marshal, err := json.Marshal(user)
	checkError(err)
	log.Printf("%s\n", marshal)
}
func insertGetId() int64 {
	var user = User{
		Name:  "a",
		Email: "b",
	}
	lastId, err := db().InsertGetId(&user)
	checkError(err)
	checkLastInsertId(lastId)
	to()
	return lastId
}
func update() {
	var user = User{
		Id:    33,
		Name:  "a2",
		Email: strconv.Itoa(int(time.Now().UnixMicro())),
	}
	aff, err := db().Update(&user)
	checkError(err)
	checkAffectedRows(aff)
	to()
}
func delete() {
	id := insertGetId()
	var user = User{
		Id: id,
	}
	aff, err := db().Delete(&user)
	checkError(err)
	checkAffectedRows(aff)
	to()
}

func TestDatabase_ToSql() {
	prepare, values, err := db().Table("a").Join("t", "a.id", "t.aid").Select("b").Where("c", 1).OrderBy("id").Limit(10).Page(2).ToSql()
	if err != nil {
		log.Fatal(err.Error())
	}
	log.Println(prepare)
	log.Println(values)
}

func TestDatabase_ToSql2() {
	var user User
	prepare, values, err := db().ToSqlTo(&user)
	if err != nil {
		log.Fatal(err.Error())
	}
	log.Println(prepare)
	log.Println(values)
}

func TestDatabase_ToSql3() {
	var user = User{Id: 1}
	prepare, values, err := db().ToSqlTo(&user)
	if err != nil {
		log.Fatal(err.Error())
	}
	log.Println(prepare)
	log.Println(values)
}

func TestDatabase_ToSql4() {
	var user []User
	prepare, values, err := db().ToSqlTo(&user)
	if err != nil {
		log.Fatal(err.Error())
	}
	log.Println(prepare)
	log.Println(values)
}

func TestDatabase_ToSqlInsert() {
	var user = User{Name: "john"}
	prepare, values, err := db().ToSqlInsert(&user, "", nil)
	if err != nil {
		log.Fatal(err.Error())
	}
	log.Println(prepare)
	log.Println(values)
}

func TestDatabase_ToSqlUpdate() {
	var user = User{Id: 1, Name: "john"}
	prepare, values, err := db().ToSqlUpdate(&user)
	if err != nil {
		log.Fatal(err.Error())
	}
	log.Println(prepare)
	log.Println(values)
}
func TestDatabase_ToSqlDelete() {
	var user = User{Id: 1, Name: "john"}
	prepare, values, err := db().ToSqlDelete(&user)
	if err != nil {
		log.Fatal(err.Error())
	}
	log.Println(prepare)
	log.Println(values)
}
