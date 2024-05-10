package cmdexec

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/a-schus/REST-API/internal/app/store"
)

var st store.Store

func TestMain(m *testing.M) {
	user := flag.String("n", "schus", "User name")
	pass := flag.String("p", "19schus78", "User password")
	host := flag.String("dbip", "localhost", "Database server IP")
	flag.Parse()
	conf := store.DBConf{
		User: *user,
		Pass: *pass,
		Host: *host,
		Name: "restapi_test",
	}

	fmt.Println("=== RUN   tests for cmdexec_test.go")
	st = store.Store{}
	if st.Open(conf) != nil {
		return
	}

	//Очищаем базу перед началом тестов на случай, если в предыдущий раз тесты завершились аварийно
	store.ClearDB(st.DB())
	st.InitEmptyDB()

	res := m.Run()

	store.ClearDB(st.DB())
	st.Close()

	os.Exit(res)
}

type respWriter struct {
	Resp string
}

func (rw respWriter) Header() http.Header {
	return make(http.Header)
}

func (rw respWriter) Write(b []byte) (int, error) {
	rw.Resp = string(b)
	return len(rw.Resp), nil
}

func (rw respWriter) WriteHeader(statusCode int) {}

func TestExecScript(t *testing.T) {
	var w respWriter
	ExecScript("Command 1", "echo \"TestExecScript\"", &st, w)
	row := st.DB().QueryRow("SELECT result FROM log WHERE name = $1", "Command 1")
	var res string
	row.Scan(&res)

	if res != "TestExecScript" {
		t.Errorf("Wrong result: \nwant %s, \nhave %s", "Command 1", res)
	}

	store.ClearTables(st.DB())
}

func TestExecLongScript(t *testing.T) {
	var w respWriter
	ch := make(chan bool)
	ctx, cansel := context.WithCancel(context.Background())
	defer cansel()

	go ExecLongScript(ctx, ch, 1, "Command 1", "echo \"String 1\"\necho \"String 2\"", &st, w)
	<-ch

	rows, _ := st.DB().Query("SELECT result FROM log WHERE name = $1", "Command 1")
	defer rows.Close()
	var res string
	for rows.Next() {
		var s string
		rows.Scan(&s)
		res += s
	}

	if res != "String 1\nString 2\n" {
		t.Errorf("Wrong result: \nwant %s \nhave %s", "String 1\nString 2\n", res)
	}

	store.ClearTables(st.DB())
}

func TestStop(t *testing.T) {
	var w respWriter
	ch := make(chan bool)
	ctx, cansel := context.WithCancel(context.Background())
	defer cansel()

	go ExecLongScript(ctx, ch, 1, "Command 1", "echo \"String 1\"\nsleep 3\necho \"String 2\"", &st, w)
	time.Sleep(1 * time.Second)
	go Stop(1, w)
	<-ch

	rows, _ := st.DB().Query("SELECT result FROM log WHERE name = $1", "Command 1")
	defer rows.Close()
	var res string
	for rows.Next() {
		var s string
		rows.Scan(&s)
		res += s
	}

	if res != "String 1\n" {
		t.Errorf("Wrong result: \nwant %s \nhave %s", "String 1\n", res)
	}

	store.ClearTables(st.DB())
}
