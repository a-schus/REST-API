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
	log.Println(string(p))
	if len(p) > 0 {
		l.db.WriteLog(l.commID, l.name, "", string(p))
	}
	l.Log = append(l.Log, string(p))
	return len(p), nil
}

func (l *LogWriter) String() string {
	return strings.Join(l.Log, "")
}

func ExecLongScript(ctx context.Context, doneCh chan bool, id int, name string, script string, db *store.Store, w http.ResponseWriter) {
	cmdChans.Add(id, doneCh)
	defer cmdChans.Remove(id)

	// db.WriteLog(id, name, script, "Long command is runing")
	outBuf := newLogWriter(db, id, name)
	errBuf := newLogWriter(db, id, name)
	c := exec.CommandContext(ctx, "bash", "-c", script)
	c.Stdout = outBuf
	c.Stderr = errBuf
	/*err := */ c.Run()
	// db.WriteLog(id, name, script, "Long command is done")
	doneCh <- true
	// if err != nil {
	// 	http.Error(w, err.Error(), http.StatusBadRequest)
	// } else {
	// 	d := outBuf.String()
	// 	d = d[:len(d)-1]
	// 	db.WriteLog(id, name, script, d)
	// 	w.Write([]byte(outBuf.String()))
	// }
}

func ExecScript(name string, script string, db *store.Store, w http.ResponseWriter) {
	outBuf := new(strings.Builder)
	c := exec.Command("bash", "-c", script)
	c.Stdout = outBuf
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

// type ctxId struct {
// 	mut      sync.Mutex
// 	cmdCtx map[int]context.Context
// }

// var cmdCtx = NewCtxId()

// func NewCtxId() ctxId {
// 	ctx := make(map[int]context.Context)
// 	return ctxId{
// 		cmdCtx: ctx,
// 	}
// }

// func (c *ctxId) Add(ctx context.Context, id int) bool {
// 	res := true
// 	c.mut.Lock()
// 	if _, ok := c.cmdCtx[id]; !ok {
// 		res = false
// 	}
// 	c.cmdCtx[id] = ctx
// 	c.mut.Unlock()

// 	return res
// }

// func (c *ctxId) Remove(id int) bool {
// 	res := true
// 	c.mut.Lock()
// 	if _, ok := c.cmdCtx[id]; !ok {
// 		res = false
// 	} else {
// 		delete(c.cmdCtx, id)
// 	}
// 	c.mut.Unlock()

// 	return res
// }

// func Stop(id int, w http.ResponseWriter) {
// 	if ctx, ok := cmdCtx.cmdCtx[id]; ok {
// 		ctx.Done()
// 		w.Write([]byte("Long command ID " + fmt.Sprint(id) + " stoped\n"))
// 	} else {
// 		w.Write([]byte("Long command ID " + fmt.Sprint(id) + " not runing\n"))
// 		log.Printf("exec: Long command ID %d not runing\n", id)
// 	}
// }

// func Exec(cmd string, w http.ResponseWriter) {
// 	// Если команда короткая, просто выполняем ее и отправляем результат
// 	out, err := exec.Command("bash", "-c", cmd).Output()
// 	if err != nil {
// 		log.Println(err.Error())
// 		http.Error(w, "Bad command", http.StatusBadRequest)
// 	} else {
// 		log.Printf("%s\n%s\n", cmd, out)
// 		// fmt.Fprintln(w, string(out))
// 		w.Write(out)
// 	}
// }

// func ExecLong(name string, cmds pq.StringArray, db *store.Store, w http.ResponseWriter, ch chan bool) {
// 	id := db.GetNextID()
// 	w.Write([]byte("Long command is running. Command ID " + fmt.Sprint(id) + "\n"))
// 	ch <- true // разлочиваем вызывающую функцию

// 	log.Printf("Long command is running. Command ID %d\n", id)
// 	db.WriteLog(id, name, "", "Long command is running")

// 	stopCh, _ := cmdChans.Add(id)
// 	exit := false

// 	for i, cmd := range cmds {
// 		select {
// 		// Проверяем не поступила ли команда остановить выполнение задачи
// 		case <-stopCh:
// 			log.Printf("Long command ID %d is stoped", id)
// 			db.WriteLog(id, name, cmd, "Stoped by user")
// 			cmdChans.Remove(id) // удаляем команду из списка запущенных
// 			exit = true

// 			// Выполняем очередную команду из списка
// 		default:
// 			if i == len(cmds)-1 {
// 				cmdChans.Remove(id) // удаляем команду из списка запущенных
// 			}
// 			time.Sleep(10 * time.Second)
// 			out, err := exec.Command("bash", "-c", cmd).Output()
// 			if err != nil {
// 				log.Printf("Long command ID %d Error: %s", id, err.Error())
// 				db.WriteLog(id, name, cmd, "Error: "+err.Error())
// 				exit = true
// 			}
// 			res := strings.ReplaceAll(string(out), "\n", "\t")
// 			log.Printf("Long command ID %d '%s'\n \t%s", id, cmd, out)
// 			db.WriteLog(id, name, cmd, res)
// 		}
// 		if exit {
// 			break
// 		}
// 	}
// 	if !exit {
// 		log.Printf("exec: Long command ID %d is done", id)
// 		db.WriteLog(id, name, "", "Long command is done")
// 	}
// }

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
