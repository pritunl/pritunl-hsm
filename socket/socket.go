package socket

import (
	"encoding/json"
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

			go func() {
				if r := recover(); r != nil {
					logrus.WithFields(logrus.Fields{
						"error": errors.New(fmt.Sprintf("%s", r)),
					}).Error("socket: Message handle error")
				}

				msgId, msgType, data, e := authority.UnmarshalPayload(
					s.Token, s.Secret, message)
				if e != nil {
					logrus.WithFields(logrus.Fields{
						"error": e,
					}).Error("socket: Unmarshal payload error")
					return
				}

				if msgType == "ssh_certificate" && data != nil {
					sshReq := &authority.SshRequest{}

					err = json.Unmarshal(data, sshReq)
					if err != nil {
						err = &errortypes.ParseError{
							errors.Wrap(err,
								"socket: Failed to unmarshal payload data"),
						}
						return
					}

					cert, e := authority.Sign(s.Serial, sshReq)
					if e != nil {
						logrus.WithFields(logrus.Fields{
							"error": e,
						}).Error("socket: Sign payload error")
						return
					}

					data := &authority.SshResponse{
						Certificate: cert,
					}

					resp, e := authority.MarshalPayload(msgId, s.Token,
						s.Secret, "ssh_certificate", data)
					if e != nil {
						logrus.WithFields(logrus.Fields{
							"error": e,
						}).Error("socket: Marshal payload error")
						return
					}

					queue <- resp
				}
			}()
		}
	}()

	ticker := time.NewTicker(pingInterval)
	defer ticker.Stop()

	statusTicker := time.NewTicker(statusInterval)
	defer statusTicker.Stop()

	for {
		select {
		case msg, ok := <-queue:
			if !ok {
				conn.WriteControl(websocket.CloseMessage, []byte{},
					time.Now().Add(writeTimeout))
				return
			}

			conn.SetWriteDeadline(time.Now().Add(writeTimeout))
			err = conn.WriteJSON(msg)
			if err != nil {
				return
			}
		case <-ticker.C:
			err = conn.WriteControl(websocket.PingMessage, []byte{},
				time.Now().Add(writeTimeout))
			if err != nil {
				return
			}
		case <-statusTicker.C:
			payload, e := authority.GetStatusPayload(
				s.Token, s.Secret, s.Serial)
			if e != nil {
				err = e
				return
			}

			queue <- payload
		case e := <-errChan:
			err = e
			return
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
