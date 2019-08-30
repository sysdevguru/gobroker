package signalman

import (
	"fmt"
	"os"
	"os/signal"
	"runtime/pprof"
	"sync"
	"syscall"

	"github.com/alpacahq/gopaca/log"
)

type SignalHandler func() error

var (
	handlers = map[string]SignalHandler{}
	mu       sync.RWMutex
	Done     = make(chan interface{})
)

func Wait() {
	<-Done
}

func RegisterFunc(name string, f SignalHandler) {
	log.Debug("register graceful termination", "name", name)
	handlers[name] = f
}

func Start() {
	sigChannel := make(chan os.Signal, 1)

	signal.Notify(sigChannel, syscall.SIGUSR1, syscall.SIGTERM, syscall.SIGKILL, os.Interrupt)

	go func() {
		for {
			sig := <-sigChannel
			switch sig {
			case syscall.SIGTERM:
				for name, handler := range handlers {
					if err := handler(); err != nil {
						log.Error("failed to graceful terminate", "error", err, "handler", name)
					} else {
						log.Debug("gracefully terminating", "handler", name)
					}
				}
				log.Info("gracefully terminated")
				Done <- 0
				return
			case syscall.SIGUSR1:
				fmt.Println("dumping stack traces due to SIGUSR1 request")
				pprof.Lookup("goroutine").WriteTo(os.Stdout, 1)
			default:
				log.Info("forcibly terminated")
				os.Exit(1)
			}
		}
	}()
}
