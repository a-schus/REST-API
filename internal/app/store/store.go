package store

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/lib/pq"
)

var databaseURL = "user=postgres password=1234 host=localhost dbname=restapi_dev sslmode=disable"
var defaultDatabaseURL = "user=postgres password=1234 host=localhost dbname=postgres sslmode=disable"

type DBConf struct {
	User string
	Pass string
	Host string
	Name string
}

// var conf DBConf

type Store struct {
	db *sql.DB
}

// func init() {
// 	conf = DBConf{
// 		user: "schus",
// 		pass: "19schus78",
// 		host: "localhost",
// 		name: "restapi_dev",
// 	}
// }

func (s *Store) Open(conf DBConf) error {
	db, _ := sql.Open("postgres", "user="+conf.User+" password="+conf.Pass+" host="+conf.Host+" dbname="+conf.Name+" sslmode=disable")

	err := db.Ping()
	if err != nil {
		fmt.Printf("DB open error: %v\n", err)
		if fmt.Sprintf("%s", err) == "pq: database \""+conf.Name+"\" does not exist" {
			fmt.Println("Create empty database \"" + conf.Name + "\" and try again")
		}
		return err
	}
	fmt.Println("DB open: OK!")

	s.db = db

	if err = s.InitEmptyDB(); err != nil {
		return err
	}

	fmt.Println("DB is correct.")

	return nil
}

func (s *Store) DB() *sql.DB {
	return s.db
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) InitEmptyDB() error {
	_, err := s.db.Exec("CREATE TABLE IF NOT EXISTS Commands (id integer PRIMARY KEY GENERATED BY DEFAULT AS IDENTITY, name varchar(30) NOT NULL UNIQUE, description text, command text);")
	if err != nil {
		fmt.Printf("CREATE TABLE 'Commands' error: %v\n", err)
		return err
	}

	_, err = s.db.Exec("CREATE SEQUENCE IF NOT EXISTS comm_id start 1")
	if err != nil {
		fmt.Printf("CREATE SEQUENCE error: %v\n", err)
		return err
	}

	_, err = s.db.Exec("CREATE TABLE IF NOT EXISTS Log (id integer PRIMARY KEY GENERATED BY DEFAULT AS IDENTITY, time timestamp, comm_id integer NOT NULL, name text, command text, result text);")
	if err != nil {
		fmt.Printf("CREATE TABLE 'Log' error: %v\n", err)
		return err
	}

	return nil
}
func (s *Store) GetAllCommands() (commandsList []string, e error) {
	var res []string
	rows, err := s.db.Query("SELECT name, description FROM Commands")
	if err != nil {
		return res, err
	}
	defer rows.Close()

	var id int = 1
	for rows.Next() {
		var name string
		var desc string

		rows.Scan(&name, &desc)

		res = append(res, fmt.Sprintf("%d\t", id)+name+"\t"+desc+"\n\n")
		id++
	}
	return res, nil
}

func (s *Store) GetCommand(name string) (description string, commands string, e error) {
	var desc string
	var cmd string

	row := s.db.QueryRow("SELECT description, command FROM Commands WHERE name = $1", name)
	err := row.Scan(&desc, &cmd)

	if err != nil {
		return "", "", err
	}

	return desc, cmd, nil
}

func (s *Store) NewCommand(name string, desc string, cmds string) error {
	var nullName = sql.NullString{
		String: name,
		Valid:  name != "",
	}
	_, err := s.db.Exec("INSERT INTO Commands (name, description, command) VALUES ($1, $2, $3)", nullName, desc, cmds)
	return err
}

func (s *Store) GetNextID() int {
	rows, _ := s.db.Query("SELECT nextval('comm_id')")
	defer rows.Close()
	rows.Next()
	var id int
	rows.Scan(&id)
	return id
}

func (s *Store) WriteLog(commID int, name string, cmd string, res string) error {
	now := pq.NullTime{
		Time:  time.Now(),
		Valid: true,
	}
	_, err := s.db.Exec("INSERT INTO log (time, comm_id, name, command, result) VALUES ($1, $2, $3, $4, $5)", now, commID, name, cmd, res)
	if err != nil {
		fmt.Printf("Write LOG error: %s\n", err.Error())
		return err
	}
	return nil
}

// clearDB() используется для тестирования
func ClearDB(db *sql.DB) {
	db.Exec("DROP TABLE IF EXISTS Commands;")
	db.Exec("DROP TABLE IF EXISTS Log;")
	db.Exec("DROP SEQUENCE IF EXISTS comm_id;")
}

// clearTables() используется для тестирования
func ClearTables(db *sql.DB) {
	db.Exec("DELETE FROM Commands;")
	db.Exec("DELETE FROM Log;")
	db.Exec("ALTER SEQUENCE comm_id RESTART WITH 1;")
}
