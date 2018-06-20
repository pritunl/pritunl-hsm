package socket

import (
	"fmt"
	"github.com/Sirupsen/logrus"
	"github.com/dropbox/godropbox/errors"
	"github.com/gorilla/websocket"
	"github.com/pritunl/pritunl-hsm/authority"
	"github.com/pritunl/pritunl-hsm/errortypes"
	"time"
)

type Socket struct {
	Serial string
	Token  string
	Secret string
	Host   string
}

func (s *Socket) stream() (err error) {
	conn, _, err := websocket.DefaultDialer.Dial(
		fmt.Sprintf("wss://%s/hsm", s.Host), nil)
	if err != nil {
		err = &errortypes.ParseError{
			errors.Wrap(err, "authority: Failed to connect to pritunl host"),
		}
		return
	}
	defer conn.Close()

	queue := make(chan *authority.HsmPayload, 50)
	defer close(queue)

	errChan := make(chan error, 1)

	go func() {
		defer func() {
			recover()
		}()
		for {
			_, message, e := conn.ReadMessage()
			if e != nil {
				errChan <- e
				return
			}

			e = authority.SignPayload(s.Secret, s.Serial, message)
			if e != nil {
				logrus.WithFields(logrus.Fields{
					"error": e,
				}).Error("socket: Sign payload error")
			}
		}
	}()

	ticker := time.NewTicker(pingInterval)
	defer ticker.Stop()

	for {
		select {
		//case msg, ok := <-sub:
		//	if !ok {
		//		conn.WriteControl(websocket.CloseMessage, []byte{},
		//			time.Now().Add(writeTimeout))
		//		return
		//	}
		//
		//	conn.SetWriteDeadline(time.Now().Add(writeTimeout))
		//	err = conn.WriteJSON(msg)
		//	if err != nil {
		//		return
		//	}
		case <-ticker.C:
			err = conn.WriteControl(websocket.PingMessage, []byte{},
				time.Now().Add(writeTimeout))
			if err != nil {
				return
			}
		}
	}
}

func (s *Socket) Run() {
	for {
		err := s.stream()
		if err != nil {
			logrus.WithFields(logrus.Fields{
				"error": err,
			}).Error("socket: Socket stream error")
		}

		time.Sleep(100 * time.Millisecond)
	}
}
