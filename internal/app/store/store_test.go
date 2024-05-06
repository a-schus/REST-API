package store

import (
	"database/sql"
	"os"
	"testing"
)

var store Store

func clearDB(db *sql.DB) {
	db.Exec("DROP TABLE IF EXISTS Commands;")
	db.Exec("DROP TABLE IF EXISTS Log;")
	db.Exec("DROP SEQUENCE IF EXISTS comm_id;")
}

func clearTables(db *sql.DB) {
	db.Exec("DELETE FROM Commands;")
	db.Exec("DELETE FROM Log;")
	db.Exec("ALTER SEQUENCE comm_id RESTART WITH 1;")
}

func TestMain(m *testing.M) {
	store = Store{}
	conf.name = "restapi_test"
	if store.Open() != nil {
		return
	}

	clearTables(store.db) //на всякий случай очищаем таблицы в тестовой базе

	res := m.Run()

	clearDB(store.db)

	os.Exit(res)
}

func TestNewCommand(t *testing.T) {
	type cmd struct {
		name string
		desc string
		cmds string
	}

	testCases := []struct {
		ok    bool
		input cmd
	}{
		{true, cmd{
			name: "command 1",
			desc: "description 1",
			cmds: "echo \"script 1\"",
		}},
		{false, cmd{
			name: "",
			desc: "description 2",
			cmds: "echo \"script 2\"",
		}},
		{false, cmd{
			name: "command 1",
			desc: "description 3",
			cmds: "echo \"script 3\"",
		}},
	}

	for _, tCase := range testCases {
		err := store.NewCommand(tCase.input.name, tCase.input.desc, tCase.input.cmds)

		if err == nil {
			var out cmd
			row := store.db.QueryRow("SELECT name, description, command FROM Commands WHERE name = $1", tCase.input.name)
			row.Scan(&out.name, &out.desc, &out.cmds)
			if tCase.input != out {
				t.Errorf("Wrong result: %v", tCase.input)
			}
		} else if tCase.ok {
			t.Errorf("Wrong Ok status: %v. Want %v, have %v", tCase.input, tCase.ok, err == nil)
		}
	}
}
