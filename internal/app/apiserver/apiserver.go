package apiserver

import (
	"context"
	"errors"
	"fmt"
	"io"
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
	mux.HandleFunc("/new", s.newScriptHandler)
	mux.HandleFunc("/cmd", s.cmdHandler)
	mux.HandleFunc("/exec", s.execHandler)
	mux.HandleFunc("/execlong", s.execLongHandler)
	mux.HandleFunc("/stop", s.stopHandler)
	mux.HandleFunc("/shutdown", s.shutdownHandler)

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
	// и пользовательский сигнал запроса /shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGUSR1)
	<-sigChan

	shutdownCtx, shutdownRelease := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownRelease()

	if err := s.server.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("%v", err)
	}
}

func (s *APIServer) shutdownHandler(w http.ResponseWriter, req *http.Request) {
	// log.Printf("Received %v request", req.RequestURI)
	pid := os.Getpid()
	proc, _ := os.FindProcess(pid)
	proc.Signal(syscall.SIGUSR1)
	// io.WriteString(w, "APIServer: Server stoped")
	w.Write([]byte("Server stoped\n"))
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
		} else {
			w.Header().Set("Content-Type", "text/plain")
			for _, row := range response {
				w.Write([]byte(row))
			}
		}
	} else {
		// если URL с параметром
		if response, cmds, err := s.db.GetCommand(command); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		} else {
			t := strings.Builder{}
			t.WriteString("\n")
			for range response {
				t.WriteString("-")
			}
			t.WriteString("\n")

			response += t.String() + cmds
			w.Write([]byte(response + "\n"))
		}
	}
}

func (s *APIServer) newScriptHandler(w http.ResponseWriter, req *http.Request) {

	/*
		Формат POST запроса curl, содержащего файл bash-скрипта
		curl -X POST http://localhost:8080/new -F File=@/home/schus/go/src/github.com/a-schus/REST-API/scr.sh -F name=7 -F desc=Com+Desc
	*/

	var name, desc, cmd string

	if err := req.ParseMultipartForm(10240); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	name = req.FormValue("name")
	desc = req.FormValue("desc")

	if file, _, err := req.FormFile("File"); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	} else {
		binFile, err := io.ReadAll(file)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		cmd = string(binFile)
	}

	err := s.db.NewCommand(name, desc, cmd)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	} else {
		w.Write([]byte("Command added and runing\n--------------------------\n"))
		cmdexec.ExecScript(name, cmd, s.db, w)
		w.Write([]byte("--------------------------\nCommand is done\n"))
	}
}

func (s *APIServer) execHandler(w http.ResponseWriter, req *http.Request) {
	params, _ := url.ParseQuery(req.URL.RawQuery)
	name := params.Get("name")
	_, cmd, err := s.db.GetCommand(name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	} else {
		cmdexec.ExecScript(name, cmd, s.db, w)
	}
}

func (s *APIServer) execLongHandler(w http.ResponseWriter, req *http.Request) {
	id := s.db.GetNextID()
	w.Write([]byte("Long command ID " + fmt.Sprint(id) + " is runing\n"))
	go func() {
		params, _ := url.ParseQuery(req.URL.RawQuery)
		name := params.Get("name")
		_, cmd, err := s.db.GetCommand(name)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
		} else {
			ctx, cansel := context.WithCancel(context.Background())
			defer cansel()

			doneCh := make(chan bool, 1)
			s.db.WriteLog(id, name, cmd, "Long command "+fmt.Sprintf("%d", id)+" is runing")

			go cmdexec.ExecLongScript(ctx, doneCh, id, name, cmd, s.db, w)
			ok := <-doneCh
			if ok {
				s.db.WriteLog(id, name, cmd, "Long command "+fmt.Sprintf("%d", id)+" is done")
			} else {
				s.db.WriteLog(id, name, cmd, "Long command "+fmt.Sprintf("%d", id)+" stoped by user")
			}
		}
	}()
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
