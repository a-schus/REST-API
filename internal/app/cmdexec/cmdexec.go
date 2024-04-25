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

	for _, cmd := range cmds {
		select {
		// Проверяем не поступила ли команда остановить выполнение задачи
		case <-stopCh:
			log.Printf("Long command ID %d is stoped", id)
			db.WriteLog(id, name, cmd, "Stoped by user")
			exit = true

		// Выполняем очередную команду из списка
		default:
			time.Sleep(10 * time.Second)
			out, err := exec.Command("bash", "-c", cmd).Output()
			if err != nil {
				log.Printf("Long command ID %d Error: %s", id, err.Error())
				db.WriteLog(id, name, cmd, "Error: "+err.Error())
				exit = true
			}
			res := strings.ReplaceAll(string(out), "\n", "\t")
			// log.Printf("Long command ID %d '%s'\n \t%s", id, cmd, out)
			db.WriteLog(id, name, cmd, res)
		}
		if exit {
			break
		}
	}
	// close(cmdChans.cmdChans[id])
	cmdChans.Remove(id) // удаляем команду из списка запущенных
	if !exit {
		log.Printf("exec: Long command ID %d is done", id)
		db.WriteLog(id, name, "", "Long command is done")

	}
}

// func NextID() int {
// 	return 1
// }

// Останавливает команду по переданному ID
func Stop(id int, w http.ResponseWriter) {
	if ch, ok := cmdChans.cmdChans[id]; ok {
		ch <- true
		// io.WriteString(w, "Long command ID "+fmt.Sprint(id)+" stoped\n")
		w.Write([]byte("Long command ID " + fmt.Sprint(id) + " stoped\n"))

		// cmdChans.Remove(id)
	} else {
		// io.WriteString(w, "Long command ID "+fmt.Sprint(id)+" not runing\n")
		w.Write([]byte("Long command ID " + fmt.Sprint(id) + " not runing\n"))
		log.Printf("exec: Long command ID %d not runing\n", id)
	}
}

// Структура для хранения ID и канала для каждой долгой команды
type chanId struct {
	mut      sync.Mutex
	cmdChans map[int]chan (bool)
}

var cmdChans = NewChanId()

func NewChanId() chanId {
	return chanId{
		cmdChans: make(map[int]chan (bool)),
	}
}

func (c *chanId) Add(id int) (chan (bool), bool) {
	c.mut.Lock()
	if _, err := c.cmdChans[id]; err {
		return nil, false
	}
	c.cmdChans[id] = make(chan bool)
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
