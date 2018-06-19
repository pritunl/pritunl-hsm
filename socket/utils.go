package socket

import (
	"github.com/pritunl/pritunl-hsm/config"
	"net/url"
	"strings"
)

func New(uri string) (sock *Socket) {
	u, err := url.Parse(uri)
	if err != nil || u.User == nil {
		return
	}

	pass, _ := u.User.Password()

	sock = &Socket{
		Serial: strings.TrimLeft(u.Path, "/"),
		Token:  u.User.Username(),
		Secret: pass,
		Host:   u.Host,
	}

	return
}

func Init() (err error) {
	for _, uri := range config.Config.PritunlZeroHosts {
		sock := New(uri)
		if sock != nil {
			go sock.Run()
		}
	}

	return
}
