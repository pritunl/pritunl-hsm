package utils

import (
	"bytes"
	"encoding/base64"
	"golang.org/x/crypto/ssh"
)

func MarshalCertificate(cert *ssh.Certificate, comment string) []byte {
	b := &bytes.Buffer{}
	b.WriteString(cert.Type())
	b.WriteByte(' ')
	e := base64.NewEncoder(base64.StdEncoding, b)
	e.Write(cert.Marshal())
	e.Close()
	if comment != "" {
		b.WriteByte(' ')
		b.Write([]byte(comment))
	}
	return b.Bytes()
}

func MarshalPublicKey(key ssh.PublicKey) []byte {
	b := &bytes.Buffer{}
	b.WriteString(key.Type())
	b.WriteByte(' ')
	e := base64.NewEncoder(base64.StdEncoding, b)
	e.Write(key.Marshal())
	e.Close()
	return b.Bytes()
}
