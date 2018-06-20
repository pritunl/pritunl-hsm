package main

import (
	"github.com/Sirupsen/logrus"
	"github.com/pritunl/pritunl-hsm/config"
	"github.com/pritunl/pritunl-hsm/constants"
	"github.com/pritunl/pritunl-hsm/logger"
	"github.com/pritunl/pritunl-hsm/socket"
	"github.com/pritunl/pritunl-hsm/yubikey"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	err := config.Init()
	if err != nil {
		panic(err)
	}

	logger.Init()

	err = yubikey.Init()
	if err != nil {
		panic(err)
	}

	logrus.Info("main: Starting sockets")

	err = socket.Init()
	if err != nil {
		panic(err)
	}

	sig := make(chan os.Signal, 2)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	<-sig

	constants.Interrupt = true

	logrus.Info("main: Shutting down")
	time.Sleep(300 * time.Millisecond)
}
