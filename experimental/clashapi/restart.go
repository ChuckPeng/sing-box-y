package clashapi

import (
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"syscall"

	"github.com/sagernet/sing-box/adapter"
	"github.com/sagernet/sing-box/log"
	"github.com/sagernet/sing/common"
	"github.com/sagernet/sing/service"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

func restartRouter(server *Server, logFactory log.Factory) http.Handler {
	r := chi.NewRouter()
	r.Post("/", restart(server, logFactory))
	return r
}

func restart(server *Server, logFactory log.Factory) http.HandlerFunc {
	restartExecutable := func(execPath string) {
		inbound := service.FromContext[adapter.InboundManager](server.ctx)
		common.Close(
			inbound, server.router.FakeIPStore(),
		)
		logger := logFactory.Logger()
		logger.Info("sing-box restarting")
		if runtime.GOOS == "windows" {
			cmd := exec.Command(execPath, os.Args[1:]...)
			cmd.Stdin = os.Stdin
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			err := cmd.Start()
			if err != nil {
				logger.Error("sing-box restarting: ", err)
			}

			os.Exit(0)
		}

		err := syscall.Exec(execPath, os.Args, os.Environ())
		if err != nil {
			logger.Error("sing-box restarting: ", err)
		}
	}
	return func(w http.ResponseWriter, r *http.Request) {
		execPath, err := os.Executable()
		if err != nil {
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, newError(err.Error()))
			return
		}

		render.JSON(w, r, render.M{"status": "ok"})

		go restartExecutable(execPath)
	}
}
