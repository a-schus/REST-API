package cmdexec

import (
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/a-schus/REST-API/internal/app/store"
	"github.com/lib/pq"
)

func Exec(cmd string, w http.ResponseWriter) {
	// Если команда короткая, просто выполняем ее и отправляем результат
	out, err := exec.Command("bash", "-c", cmd).Output()
	if err != nil {
		log.Println(err.Error())
		http.Error(w, "Bad command", http.StatusBadRequest)
	} else {
		log.Printf("%s\n%s\n", cmd, out)
		// fmt.Fprintln(w, string(out))
		w.Write(out)
	}
}

func ExecLong(name string, cmds pq.StringArray, db *store.Store, w http.ResponseWriter, ch chan bool) {
	id := db.GetNextID()
	w.Write([]byte("Long command is running. Command ID " + fmt.Sprint(id) + "\n"))
	ch <- true // разлочиваем вызывающую функцию

	log.Printf("Long command is running. Command ID %d\n", id)
	db.WriteLog(id, name, "", "Long command is running")

	stopCh, _ := cmdChans.Add(id)
	exit := false

	for i, cmd := range cmds {
		select {
		// Проверяем не поступила ли команда остановить выполнение задачи
		case <-stopCh:
			log.Printf("Long command ID %d is stoped", id)
			db.WriteLog(id, name, cmd, "Stoped by user")
			cmdChans.Remove(id) // удаляем команду из списка запущенных
			exit = true

			// Выполняем очередную команду из списка
		default:
			if i == len(cmds)-1 {
				cmdChans.Remove(id) // удаляем команду из списка запущенных
			}
			time.Sleep(10 * time.Second)
			out, err := exec.Command("bash", "-c", cmd).Output()
			if err != nil {
				log.Printf("Long command ID %d Error: %s", id, err.Error())
				db.WriteLog(id, name, cmd, "Error: "+err.Error())
				exit = true
			}
			res := strings.ReplaceAll(string(out), "\n", "\t")
			log.Printf("Long command ID %d '%s'\n \t%s", id, cmd, out)
			db.WriteLog(id, name, cmd, res)
		}
		if exit {
			break
		}
	}
	if !exit {
		log.Printf("exec: Long command ID %d is done", id)
		db.WriteLog(id, name, "", "Long command is done")
	}
}

// Останавливает команду по переданному ID
func Stop(id int, w http.ResponseWriter) {
	if ch, ok := cmdChans.cmdChans[id]; ok {
		ch <- true
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

func (c *chanId) Add(id int) (chan bool, bool) {
	c.mut.Lock()
	if _, err := c.cmdChans[id]; err {
		return nil, false
	}
	c.cmdChans[id] = make(chan bool, 1)
	c.mut.Unlock()

	return c.cmdChans[id], true
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
