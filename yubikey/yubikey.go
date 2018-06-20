package yubikey

import (
	"github.com/dropbox/godropbox/errors"
	"github.com/pritunl/pritunl-hsm/errortypes"
	"github.com/pritunl/pritunl-hsm/utils"
	"github.com/pritunl/pritunl-hsm/ykpiv"
	"golang.org/x/crypto/ssh"
)

var (
	keys     = map[string]*ykpiv.Yubikey{}
	pubKeys  = map[string]string{}
	keysLock = utils.NewMultiLock()
)

func GetKey(serial string) (key *ykpiv.Yubikey) {
	// TODO
	for _, key = range keys {
		return
	}

	return
}

func GetPublicKey(serial string) (pubKey string) {
	// TODO
	for _, pubKey = range pubKeys {
		return
	}

	return
}

func LockKey(serial string) {
	keysLock.Lock(serial)
}

func UnlockKey(serial string) {
	keysLock.Unlock(serial)
}

func Init() (err error) {
	ks := map[string]*ykpiv.Yubikey{}
	pks := map[string]string{}

	// TODO
	pin := "123456"

	yubikey, err := ykpiv.New(ykpiv.Options{
		Reader: "Yubico Yubikey 4 OTP+U2F+CCID 00 00",
		PIN:    &pin,
	})
	if err != nil {
		return
	}

	err = yubikey.Login()
	if err != nil {
		return
	}

	slot, err := yubikey.Authentication()
	if err != nil {
		return
	}

	pubKey, err := ssh.NewPublicKey(slot.Public())
	if err != nil {
		err = &errortypes.ParseError{
			errors.Wrap(err, "yubikery: Failed to parse rsa key"),
		}
		return
	}

	ks["todo"] = yubikey
	pks["todo"] = string(utils.MarshalPublicKey(pubKey))

	keys = ks

	return
}
