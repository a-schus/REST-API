package apiserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/a-schus/REST-API/internal/app/cmdexec"
	"github.com/a-schus/REST-API/internal/app/store"
	"github.com/lib/pq"
)

type APIServer struct {
	server *http.Server
	db     *store.Store
}

func New(ip string, _db *store.Store) *APIServer {
	return &APIServer{
		server: &http.Server{
			Addr: ip,
		},
		db: _db,
	}
}

func (s *APIServer) Start() {
	mux := http.NewServeMux()
	mux.HandleFunc("/close", s.closeHandler)
	mux.HandleFunc("/stop", s.stopHandler)
	mux.HandleFunc("/cmd", s.cmdHandler)
	mux.HandleFunc("/new", s.newHandler)
	mux.HandleFunc("/exec", s.execHandler)

	s.server.Handler = mux

	// Запускаем прослушивание порта
	go func() {
		if err := s.server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("%v", err)
		}
		log.Println("Server stoped")
	}()

	log.Printf("Server started")
	log.Printf("The IP address being listened to %s\n", s.server.Addr)

	// Мониторим системные сигналы на завершение программы
	// и пользовательский сигнал запроса /close
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGUSR1)
	<-sigChan

	shutdownCtx, shutdownRelease := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownRelease()

	if err := s.server.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("%v", err)
	}
}

func (s *APIServer) closeHandler(w http.ResponseWriter, req *http.Request) {
	log.Printf("Received %v request", req.RequestURI)
	pid := os.Getpid()
	proc, _ := os.FindProcess(pid)
	proc.Signal(syscall.SIGUSR1)
	// io.WriteString(w, "APIServer: Server stoped")
	w.Write([]byte("Server stoped\n"))
}

func (s *APIServer) stopHandler(w http.ResponseWriter, req *http.Request) {
	params, _ := url.ParseQuery(req.URL.RawQuery)
	// var id int
	id, err := strconv.Atoi(params.Get("id"))

	log.Printf("Received %v request", req.RequestURI)
	if err != nil {
		log.Println(err.Error())
	} else {
		cmdexec.Stop(id, w)
	}
}

// Обработчик запроса 'cmd'
// В зависимости от наличия или отсутствия параметров возвращает в ответ
// список всех команд с описанием или описание и содержимое запрошенной команды
func (s *APIServer) cmdHandler(w http.ResponseWriter, req *http.Request) {
	params, _ := url.ParseQuery(req.URL.RawQuery)
	command := params.Get("name")

	if command == "" {
		// если URL без параметров
		if response, err := s.db.GetAllCommands(); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			// w.Write([]byte("Status: " + fmt.Sprintf("%d", http.StatusInternalServerError) + " Internal Server Error"))
			// w.Write([]byte(err.Error()))
			// w.WriteHeader(http.StatusInternalServerError)
		} else {
			w.Header().Set("Content-Type", "text/plain")
			for _, row := range response {
				w.Write([]byte(row))
			}
			// w.WriteHeader(http.StatusOK)
		}
	} else {
		// если URL с параметром
		if response, cmds, err := s.db.GetCommand(command); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			// w.Write([]byte("Status: " + fmt.Sprintf("%d", http.StatusInternalServerError) + " Internal Server Error\n"))
			// w.Write([]byte(err.Error()))
			// w.WriteHeader(http.StatusInternalServerError)
		} else {
			response += ":\t"
			for _, cmd := range cmds {
				response += cmd + "; "
			}
			w.Write([]byte(response + "\n"))
			// w.WriteHeader(http.StatusOK)
		}
	}
}

type Args struct {
	Name string `json:"name"`
	Desc string `json:"desc"`
	Cmd  string `json:"cmd"`
}

func (s *APIServer) newHandler(w http.ResponseWriter, req *http.Request) {

	/* Формат POST запроса curl, содержащего новую команду в теле
	curl -X POST -json -d '{
		"name": "date",
		"desc": "Show current date",
		"cmd": "echo \"Current date: \";date"
	}' http://localhost:8080/new
	*/

	params, _ := url.ParseQuery(req.URL.RawQuery)
	var name, desc, cmd, splitter string

	contLen := req.ContentLength
	// Если запрос содержит тело, аргументы, переданные через URL игнорируются
	if contLen > 0 {
		contByte := make([]byte, contLen)
		req.Body.Read(contByte)

		args := Args{}

		e := json.Unmarshal(contByte, &args)
		fmt.Println(e)

		name = args.Name
		desc = args.Desc
		cmd = args.Cmd
		splitter = ";"
	} else {
		name = params.Get("name")
		desc = params.Get("desc")
		cmd = params.Get("cmd")
		splitter = "@@"
	}

	cmds := pq.StringArray(strings.Split(cmd, splitter))

	err := s.db.NewCommand(name, desc, cmds)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		// w.Write([]byte(err.Error()))
	} else {
		// w.Write([]byte("Ok!"))
		w.WriteHeader(http.StatusOK)
	}
}

func (s *APIServer) execHandler(w http.ResponseWriter, req *http.Request) {
	params, _ := url.ParseQuery(req.URL.RawQuery)
	name := params.Get("name")
	_, cmd, err := s.db.GetCommand(name)

	if err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		// w.Write([]byte("Status: " + fmt.Sprintf("%d", http.StatusBadRequest) + " Bad request\n"))
		// w.Write([]byte(err.Error()))
		// w.WriteHeader(http.StatusBadRequest)
	} else {
		if len(cmd) == 1 {
			cmdexec.Exec(cmd[0], w)
		} else if len(cmd) > 1 {
			ch := make(chan (bool))
			go cmdexec.ExecLong(name, cmd, s.db, w, ch)
			<-ch // ждем, сообщения о запуске долгой команды
		}
		// w.WriteHeader(http.StatusOK)
	}
}
