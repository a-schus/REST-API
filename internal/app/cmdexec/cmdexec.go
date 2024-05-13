package cmdexec

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"strings"
	"sync"

	"github.com/a-schus/REST-API/internal/app/store"
)

type LogWriter struct {
	Log    []string
	db     *store.Store
	commID int
	name   string
}

func newLogWriter(_db *store.Store, _commID int, _name string) *LogWriter {
	return &LogWriter{
		db:     _db,
		commID: _commID,
		name:   _name,
	}
}
func (l *LogWriter) Write(p []byte) (n int, err error) {
	if len(p) > 0 {
		err = l.db.WriteLog(l.commID, l.name, "", string(p))
	}
	l.Log = append(l.Log, string(p))
	return len(p), err
}

func (l *LogWriter) String() string {
	return strings.Join(l.Log, "")
}

func ExecLongScript(ctx context.Context, doneCh chan bool, id int, name string, script string, db *store.Store, w http.ResponseWriter) {
	cmdChans.Add(id, doneCh)
	defer cmdChans.Remove(id)

	outBuf := newLogWriter(db, id, name)
	errBuf := newLogWriter(db, id, name)
	c := exec.CommandContext(ctx, "bash", "-c", script)
	c.Stdout = outBuf
	c.Stderr = errBuf
	c.Run()
	doneCh <- true
}

func ExecScript(name string, script string, db *store.Store, w http.ResponseWriter) {
	outBuf := new(strings.Builder)
	c := exec.Command("bash", "-c", script)
	c.Stdout = outBuf
	c.Stderr = outBuf
	err := c.Run()
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	} else {
		d := strings.Split(outBuf.String(), "\n")
		d = d[:len(d)-1]
		db.WriteLog(db.GetNextID(), name, script, strings.Join(d, "\n"))
		w.Write([]byte(outBuf.String()))
	}
}

// Останавливает команду по переданному ID
func Stop(id int, w http.ResponseWriter) {
	if ch, ok := cmdChans.cmdChans[id]; ok {
		ch <- false
		w.Write([]byte("Long command ID " + fmt.Sprint(id) + " stoped\n"))
	} else {
		w.Write([]byte("Long command ID " + fmt.Sprint(id) + " not runing\n"))
		log.Printf("exec: Long command ID %d not runing\n", id)
	}
}

// Структура для хранения ID и канала для каждой долгой команды
type chanId struct {
	mut      sync.Mutex
	cmdChans map[int]chan bool
}

var cmdChans = NewChanId()

func NewChanId() chanId {
	ch := make(map[int]chan bool, 1)
	return chanId{
		cmdChans: ch,
	}
}

func (c *chanId) Add(id int, ch chan bool) bool {
	res := true
	c.mut.Lock()
	if _, err := c.cmdChans[id]; err {
		res = false
	}
	c.cmdChans[id] = ch
	c.mut.Unlock()

	return res
}

func (c *chanId) Remove(id int) bool {
	res := true
	c.mut.Lock()
	if _, ok := c.cmdChans[id]; !ok {
		res = false
	} else {
		delete(c.cmdChans, id)
	}
	c.mut.Unlock()

	return res
}
