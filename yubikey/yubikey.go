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
	keysLock = utils.NewMultiLock()
)

func GetKey(serial string) (key *ykpiv.Yubikey) {
	// TODO
	// slot = slots[serial]

	for _, key = range keys {
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

	encodedPub := string(utils.MarshalPublicKey(pubKey))

	println("***************************************************")
	println(encodedPub)
	println("***************************************************")

	ks["todo"] = yubikey

	keys = ks

	return
}
