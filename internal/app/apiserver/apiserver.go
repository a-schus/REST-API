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
	log.Printf("APIServer: The IP address being listened to %s\n", ip)

	return &APIServer{
		server: &http.Server{
			Addr: ip,
		},
		db: _db,
	}
}

func (s *APIServer) Start() {
	mux := http.NewServeMux()
	mux.HandleFunc("/h", s.hHandler)
	mux.HandleFunc("/close", s.closeHandler)
	mux.HandleFunc("/date", s.dateHandler)
	mux.HandleFunc("/stop", s.stopHandler)
	mux.HandleFunc("/cmd", s.cmdHandler)
	mux.HandleFunc("/new", s.newHandler)

	s.server.Handler = mux

	// Запускаем прослушивание порта
	go func() {
		log.Printf("APIServer: Server started")

		if err := s.server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			log.Printf("%v", err)
		}
		log.Println("APIServer: Server stoped")
	}()

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

func (s *APIServer) hHandler(w http.ResponseWriter, req *http.Request) {
	log.Printf("http: Received %v request", req.RequestURI)
	io.WriteString(w, "Hello, client!\n")
}

func (s *APIServer) closeHandler(w http.ResponseWriter, req *http.Request) {
	log.Printf("http: Received %v request", req.RequestURI)
	pid := os.Getpid()
	proc, _ := os.FindProcess(pid)
	proc.Signal(syscall.SIGUSR1)
	io.WriteString(w, "APIServer: Server stoped")
}

func (s *APIServer) dateHandler(w http.ResponseWriter, req *http.Request) {
	// log.Printf("http: Received %v request", req.RequestURI)

	// // TO-DO получение и парсинг команды
	// cmd := []sql.NullString{"date", "date", "date"}

	// if len(cmd) <= 1 {
	// 	cmdexec.Exec(cmd[0], w)
	// } else {
	// 	ch := make(chan bool)
	// 	go cmdexec.ExecLong(cmd, w, ch)
	// 	<-ch // ждем сообщения, что длинная команда запущена, и отпускаем горутину
	// }
}

func (s *APIServer) stopHandler(w http.ResponseWriter, req *http.Request) {
	log.Printf("http: Received %v request", req.RequestURI)
	// TO-DO получение очередного ID
	cmdexec.Stop(1, w)
}

// Обработчик запроса 'cmd'
// В зависимости от наличия или отсутствия параметров возвращает в ответ
// список всех команд с описанием или описание и содержимое запрошенной команды
func (s *APIServer) cmdHandler(w http.ResponseWriter, req *http.Request) {
	params, _ := url.ParseQuery(req.URL.RawQuery)
	command := params.Get("cmd")

	if command == "" {
		// если URL без параметров
		if response, err := s.db.GetAllCommands(); err != nil {
			w.Write([]byte("Status: " + fmt.Sprintf("%d", http.StatusInternalServerError) + " Internal Server Error"))
			w.Write([]byte(err.Error()))
			w.WriteHeader(http.StatusInternalServerError)
		} else {
			w.Header().Set("Content-Type", "text/plain")
			w.Write([]byte(*response))
			w.WriteHeader(http.StatusOK)
		}
	} else {
		// если URL с параметром
		if response, cmds, err := s.db.GetCommand(command); err != nil {
			w.Write([]byte("Status: " + fmt.Sprintf("%d", http.StatusInternalServerError) + " Internal Server Error\n"))
			w.Write([]byte(err.Error()))
			w.WriteHeader(http.StatusInternalServerError)
		} else {
			response += ":\t"
			for _, cmd := range cmds {
				response += cmd.String + "; "
			}
			w.Header().Set("Content-Type", "text/plain")
			w.Write([]byte(response))
			w.WriteHeader(http.StatusOK)
		}
	}
}

func (s *APIServer) newHandler(w http.ResponseWriter, req *http.Request) {
	params, _ := url.ParseQuery(req.URL.RawQuery)
	name := params.Get("name")
	desc := params.Get("desc")
	cmd := params.Get("cmd")

	err := s.db.NewCommand(name, desc, cmd)
	if err != nil {
		w.Write([]byte(err.Error()))
	} else {
		w.Write([]byte("Ok!"))
		w.WriteHeader(http.StatusOK)
	}
}
