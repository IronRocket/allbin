package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/mattn/go-sqlite3"
)

var db *sql.DB

type Dimension struct {
	Id                string `json:"id"`
	Title             string `json:"title"`
	Encrypted         bool   `json:"encrypted"`
	Visibility        bool   `json:"visibility"`
	Text              string `json:"text"`
	FileName          string `json:"fileName"`
	Reads             int    `json:"reads"`
	DownloadLimit     int    `json:"downloadLimit"`
	ExpirationDate    int    `json:"expirationDate"`
	ExpirationDateISO string
}

func spaceWrite(id string, dim Dimension) error {
	var query string = `insert into InvisibleSpace (
		id, title, encrypted, fileName,
		text, downloadLimit, reads, expirationDate)
		values(?, ?, ?, ?, ?, ?, ?, ?)
		`
	if dim.Visibility {
		query = `insert into Space (
			id, title, encrypted, fileName,
			text, downloadLimit, reads, expirationDate)
			values(?, ?, ?, ?, ?, ?, ?, ?)`
	}

	stmt, err := db.Prepare(query)
	if err != nil {
		return err
	}
	defer stmt.Close()
	fmt.Println("dimWrite", dim)
	_, err = stmt.Exec(
		id, dim.Title, dim.Encrypted,
		dim.FileName, dim.Text,
		dim.DownloadLimit, dim.Reads,
		dim.ExpirationDate)
	if err != nil {
		return err
	}
	return nil
}

func main() {
	os.Remove("./database.db")
	var err error
	db, err = sql.Open("sqlite3", "./database.db")
	if err != nil {
		log.Fatal(err)
	}

	sqlStmt := `
	create table InvisibleSpace (
		id tinytext not null primary key,
		title tinytext,
		encrypted boolean,
		fileName tinytext,
		text text,
		reads integer,
		downloadLimit integer,
		expirationDate integer
	);
	create table Space (
		id tinytext not null primary key,
		title tinytext,
		encrypted boolean,
		fileName tinytext,
		text text,
		reads integer,
		downloadLimit integer,
		expirationDate integer
	);
	`
	_, err = db.Exec(sqlStmt)
	if err != nil {
		log.Printf("%q: %s\n", err, sqlStmt)
		return
	}

	dim := Dimension{
		Title:          "test",
		Encrypted:      false,
		Visibility:     true,
		Text:           "data",
		FileName:       "23.jpg",
		Reads:          0,
		DownloadLimit:  0,
		ExpirationDate: 12343214,
	}
	err = spaceWrite("432", dim)

	if err != nil {
		panic(err)
	}

}
