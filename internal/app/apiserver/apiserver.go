package apiserver

import (
	"context"
	"errors"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/a-schus/REST-API/internal/app/cmdexec"
)

type APIServer struct {
	server *http.Server
}

func New(ip string) *APIServer {
	log.Printf("APIServer: The IP address being listened to %s\n", ip)

	return &APIServer{
		server: &http.Server{
			Addr: ip,
		},
	}
}

func (s *APIServer) Start() {
	mux := http.NewServeMux()
	mux.HandleFunc("/h", s.hHandler)
	mux.HandleFunc("/close", s.closeHandler)
	mux.HandleFunc("/date", s.dateHandler)
	mux.HandleFunc("/stop", s.stopHandler)

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
	log.Printf("http: Received %v request", req.RequestURI)

	// TO-DO получение и парсинг команды
	cmd := []string{"date", "date", "date"}

	if len(cmd) <= 1 {
		cmdexec.Exec(cmd[0], w)
	} else {
		ch := make(chan bool)
		go cmdexec.ExecLong(cmd, w, ch)
		<-ch // ждем сообщения, что длинная команда запущена, и отпускаем горутину
	}
}

func (s *APIServer) stopHandler(w http.ResponseWriter, req *http.Request) {
	log.Printf("http: Received %v request", req.RequestURI)
	// TO-DO получение очередного ID
	cmdexec.Stop(1, w)
}
