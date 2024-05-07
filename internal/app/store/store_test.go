package store

import (
	"flag"
	"fmt"
	"os"
	"testing"
)

var st Store

type cmd struct {
	name string
	desc string
	cmds string
}

func TestMain(m *testing.M) {
	user := flag.String("n", "postgres", "User name")
	pass := flag.String("p", "1234", "User password")
	host := flag.String("dbip", "localhost", "Database server IP")
	flag.Parse()
	conf := DBConf{
		User: *user,
		Pass: *pass,
		Host: *host,
		Name: "restapi_test",
	}

	fmt.Println("=== RUN   tests for store.go")
	st = Store{}
	if st.Open(conf) != nil {
		return
	}

	//Очищаем базу перед началом тестов на случай, если в предыдущий раз тесты завершились аварийно
	clearDB(st.db)
	st.initEmptyDB()

	res := m.Run()

	clearDB(st.db)
	st.Close()

	os.Exit(res)
}

func TestNewCommand(t *testing.T) {
	testCases := []struct {
		ok    bool
		input cmd
	}{
		{true, cmd{
			name: "command 1",
			desc: "New command",
			cmds: "echo \"script 1\"",
		}},
		{false, cmd{
			name: "",
			desc: "Name is NULL",
			cmds: "echo \"script 2\"",
		}},
		{false, cmd{
			name: "command 1",
			desc: "Command already exists",
			cmds: "echo \"script 3\"",
		}},
	}

	for _, tCase := range testCases {
		err := st.NewCommand(tCase.input.name, tCase.input.desc, tCase.input.cmds)

		if (err == nil) != tCase.ok {
			t.Errorf("Wrong Ok status: %v. Want %v, have %v", tCase.input, tCase.ok, err == nil)
		}

		if err == nil {
			var out cmd
			row := st.db.QueryRow("SELECT name, description, command FROM Commands WHERE name = $1", tCase.input.name)
			row.Scan(&out.name, &out.desc, &out.cmds)
			if tCase.input != out {
				t.Errorf("Wrong result: %v", tCase.input)
			}
		}
	}
	clearTables(st.db)
}

func TestGetAllCommands(t *testing.T) {
	st.NewCommand("Com 1", "New command", "echo \"script 1\"")
	if _, err := st.GetAllCommands(); err != nil {
		t.Errorf("Wrong Ok status: %v", err == nil)
	}
	clearTables(st.db)
}

func TestGetCommand(t *testing.T) {
	testCases := []struct {
		ok    bool
		input cmd
	}{
		{true, cmd{
			name: "Com 1",
		}},
		{false, cmd{
			name: "com 1",
		}},
	}

	want := cmd{
		desc: "New command",
		cmds: "echo \"script 1\"",
	}

	st.NewCommand("Com 1", "New command", "echo \"script 1\"")
	for _, tCase := range testCases {
		var out cmd
		var err error
		out.desc, out.cmds, err = st.GetCommand(tCase.input.name)

		if (err == nil) != tCase.ok {
			t.Errorf("Wrong Ok status: %v. Want %v, have %v", tCase.input, tCase.ok, err == nil)
		}

		if err == nil && out != want {
			t.Errorf("Wrong result: %v", tCase.input)
		}
	}
	clearTables(st.db)
}

func TestGetNextID(t *testing.T) {
	testCases := []struct {
		ok   bool
		want int
	}{
		{true, 1},
		{true, 2},
		{false, 4},
	}
	for _, tCase := range testCases {
		if id := st.GetNextID(); (id == tCase.want) != tCase.ok {
			t.Errorf("Wrong Ok status: %v. Want %v, have %v", tCase.want, tCase.ok, id == tCase.want)
		}
	}
	clearTables(st.db)
}

func TestWriteLog(t *testing.T) {
	type Log struct {
		commID int
		name   string
		cmd    string
		res    string
	}
	wantLog := Log{
		commID: 2341,
		name:   "Test write log",
		cmd:    "echo \"Test write log\"",
		res:    "Test write log",
	}
	var outLog Log

	st.WriteLog(wantLog.commID, wantLog.name, wantLog.cmd, wantLog.res)
	row := st.db.QueryRow("SELECT comm_id, name, command, result FROM log WHERE comm_id = $1", wantLog.commID)
	row.Scan(&outLog.commID, &outLog.name, &outLog.cmd, &outLog.res)

	if wantLog != outLog {
		t.Errorf("Wrong result: \nwant %v, \nhave %v", wantLog, outLog)
	}

	clearTables(st.db)
}
