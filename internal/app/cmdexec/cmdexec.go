package cmdexec

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os/exec"
	"strings"
	"sync"

	"github.com/lib/pq"
)

func Exec(cmd string, w http.ResponseWriter /*, ch chan bool*/) {
	// Если команда короткая, просто выполняем ее и выводим результат
	out, _ := exec.Command("bash", "-c", cmd).Output()
	out = []byte(strings.ReplaceAll(string(out), "\n", "\n\t"))
	log.Printf("exec: %s\n \t%s\n", cmd, out)
	io.WriteString(w, string(out))
	io.WriteString(w, "Done")
}

func ExecLong(cmds pq.StringArray, w http.ResponseWriter, ch chan bool) {
	for i, cmd := range cmds {
		cmds[i] = strings.ReplaceAll(cmd, "\"", "'")
	}
	id := NextID()
	io.WriteString(w, "Long command is running. Command ID "+fmt.Sprint(id))
	ch <- true
	log.Printf("exec: Long command is running. Command ID %d", id)
	stopCh, _ := cmdChans.Add(id)
	exit := false

	for _, cmd := range cmds {
		select {
		// Проверяем не поступила ли команда остановить выполнение задачи
		case <-stopCh:
			log.Printf("exec: Long command ID %d is stoped", id)
			exit = true

		// Выполняем очередную команду из списка
		default:
			// time.Sleep(5 * time.Second)
			out, err := exec.Command("bash", "-c", cmd).Output()
			if err != nil {
				log.Println("exec: Error. " + err.Error())
				exit = true
			}
			out = []byte(strings.ReplaceAll(string(out), "\n", "\t"))
			log.Printf("exec: Long command ID %d '%s'\n \t%s", id, cmd, out)
		}
		if exit {
			break
		}
	}
	cmdChans.Remove(id) // удаляем команду из списка запущенных
	if !exit {
		log.Printf("exec: Long command ID %d is done", id)
	}
}
func NextID() int {
	return 1
}

// Останавливает команду по переданному ID
func Stop(id int, w http.ResponseWriter) {
	if ch, ok := cmdChans.cmdChans[id]; ok {
		ch <- true
		io.WriteString(w, "Long command ID "+fmt.Sprint(id)+" stoped")
		// cmdChans.Remove(id)
	} else {
		io.WriteString(w, "Long command ID "+fmt.Sprint(id)+" not runing")
		log.Printf("exec: Long command ID %d not runing", id)
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
	c.mut.Lock()
	if _, ok := c.cmdChans[id]; !ok {
		return false
	}
	delete(c.cmdChans, id)
	c.mut.Unlock()

	return true
}
