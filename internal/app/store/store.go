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
	user string
	pass string
	host string
	name string
}

var conf DBConf

type Store struct {
	db *sql.DB
}

func init() {
	conf = DBConf{
		user: "schus",
		pass: "19schus78",
		host: "localhost",
		name: "restapi_dev",
	}
}

func (s *Store) Open() error {
	db, _ := sql.Open("postgres", "user="+conf.user+" password="+conf.pass+" host="+conf.host+" dbname="+conf.name+" sslmode=disable")

	err := db.Ping()
	if err != nil {
		fmt.Printf("DB open error: %v\n", err)
		if fmt.Sprintf("%s", err) == "pq: database \"restapi_dev\" does not exist" {
			fmt.Println("Create empty database 'restapi_dev' and try again")
		}
		return err
	}
	fmt.Println("DB open: OK!")

	s.db = db

	if err = s.initEmptyDB(); err != nil {
		return err
	}

	fmt.Println("DB is correct.")

	return nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) initEmptyDB() error {
	_, err := s.db.Exec("CREATE TABLE IF NOT EXISTS Commands (id integer PRIMARY KEY GENERATED BY DEFAULT AS IDENTITY, name varchar(30) NOT NULL UNIQUE, description text, command text);")
	if err != nil {
		fmt.Printf("CREATE TABLE 'Commands' error: %v\n", err)
		return err
	}

	_, err = s.db.Exec("CREATE SEQUENCE IF NOT EXISTS long_comm_id start 1")
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
	_, err := s.db.Exec("INSERT INTO Commands (name, description, command) VALUES ($1, $2, $3)", name, desc, cmds)
	return err
}

func (s *Store) GetNextID() int {
	rows, _ := s.db.Query("SELECT nextval('long_comm_id')")
	defer rows.Close()
	rows.Next()
	var id int
	rows.Scan(&id)
	return id
}

func (s *Store) WriteLog(commID int, name string, cmd string, res string) {
	now := pq.NullTime{
		Time:  time.Now(),
		Valid: true,
	}
	_, err := s.db.Exec("INSERT INTO log (time, comm_id, name, command, result) VALUES ($1, $2, $3, $4, $5)", now, commID, name, cmd, res)
	if err != nil {
		fmt.Printf("Write LOG error: %s\n", err.Error())
	}
}
