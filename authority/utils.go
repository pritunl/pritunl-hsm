package authority

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/sha512"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"github.com/dropbox/godropbox/errors"
	"github.com/pritunl/pritunl-hsm/errortypes"
	"github.com/pritunl/pritunl-hsm/utils"
	"github.com/pritunl/pritunl-hsm/yubikey"
	"golang.org/x/crypto/ssh"
	"gopkg.in/mgo.v2/bson"
	"hash/fnv"
	"time"
)

func Sign(csr *SshCsr) (err error) {
	yubi := yubikey.GetKey(csr.Serial)
	if yubi == nil {
		err = &errortypes.NotFoundError{
			errors.Wrap(err, "authority: Failed to find hsm"),
		}
		return
	}

	pubKey, comment, _, _, err := ssh.ParseAuthorizedKey(
		[]byte(csr.Certificate.Key))
	if err != nil {
		err = &errortypes.ParseError{
			errors.Wrap(err, "authority: Failed to parse ssh public key"),
		}
		return
	}

	serialHash := fnv.New64a()
	serialHash.Write([]byte(bson.NewObjectId().Hex()))
	serial := serialHash.Sum64()

	validAfter := time.Now().Add(-5 * time.Minute).Unix()
	validBefore := time.Now().Add(
		time.Duration(csr.Certificate.Ttl) * time.Second).Unix()

	cert := &ssh.Certificate{
		Key:             pubKey,
		Serial:          serial,
		CertType:        csr.Certificate.CertType,
		KeyId:           csr.Certificate.KeyId,
		ValidPrincipals: csr.Certificate.ValidPrincipals,
		ValidAfter:      uint64(validAfter),
		ValidBefore:     uint64(validBefore),
		Permissions: ssh.Permissions{
			CriticalOptions: csr.Certificate.Permissions.CriticalOptions,
			Extensions:      csr.Certificate.Permissions.Extensions,
		},
	}

	yubikey.LockKey(csr.Serial)

	slot, err := yubi.Authentication()
	if err != nil {
		return
	}

	signer, err := ssh.NewSignerFromSigner(slot)
	if err != nil {
		yubikey.UnlockKey(csr.Serial)
		return
	}

	err = cert.SignCert(rand.Reader, signer)
	if err != nil {
		yubikey.UnlockKey(csr.Serial)
		return
	}

	yubikey.UnlockKey(csr.Serial)

	certMarshaled := string(utils.MarshalCertificate(cert, comment))

	println("***************************************************")
	println(certMarshaled)
	println("***************************************************")

	return
}

func SignPayload(secret, allowedSerial string, payloadJson []byte) (
	err error) {

	payload := &HsmPayload{}
	err = json.Unmarshal(payloadJson, payload)
	if err != nil {
		err = &errortypes.ParseError{
			errors.Wrap(err, "authority: Failed to unmarshal payload"),
		}
		return
	}

	if payload.Id == "" || payload.Token == "" || payload.Iv == "" ||
		payload.Data == "" {

		err = &errortypes.ParseError{
			errors.Wrap(err, "authority: Invalid payload"),
		}
		return
	}

	cipData, err := base64.StdEncoding.DecodeString(payload.Data)
	if err != nil {
		err = &errortypes.ParseError{
			errors.Wrap(err, "authority: Failed to decode payload data"),
		}
		return
	}

	hashFunc := hmac.New(sha512.New, []byte(secret))
	hashFunc.Write(cipData)
	rawSignature := hashFunc.Sum(nil)
	sig := base64.StdEncoding.EncodeToString(rawSignature)

	if subtle.ConstantTimeCompare([]byte(sig),
		[]byte(payload.Signature)) != 1 {

		err = &errortypes.AuthenticationError{
			errors.Wrap(err, "authority: Invalid signature"),
		}
		return
	}

	encKeyHash := sha256.New()
	encKeyHash.Write([]byte(secret))
	cipKey := encKeyHash.Sum(nil)

	cipIv, err := base64.StdEncoding.DecodeString(payload.Iv)
	if err != nil {
		err = &errortypes.ParseError{
			errors.Wrap(err, "authority: Failed to decode payload iv"),
		}
		return
	}

	if len(cipIv) != aes.BlockSize {
		err = &errortypes.ParseError{
			errors.Wrap(err, "authority: Invalid payload iv length"),
		}
		return
	}

	if len(cipData)%16 != 0 {
		err = &errortypes.ParseError{
			errors.Wrap(err, "authority: Invalid payload data length"),
		}
		return
	}

	block, err := aes.NewCipher(cipKey)
	if err != nil {
		err = &errortypes.ParseError{
			errors.Wrap(err, "authority: Failed to load cipher"),
		}
		return
	}

	mode := cipher.NewCBCDecrypter(block, cipIv)
	mode.CryptBlocks(cipData, cipData)
	cipData = bytes.TrimRight(cipData, "\x00")

	csr := &SshCsr{}
	err = json.Unmarshal(cipData, csr)
	if err != nil {
		err = &errortypes.ParseError{
			errors.Wrap(err, "authority: Failed to unmarshal payload data"),
		}
		return
	}

	if csr.Serial != allowedSerial {
		err = &errortypes.AuthenticationError{
			errors.Wrap(err, "authority: HSM serial mismatch"),
		}
		return
	}

	err = Sign(csr)
	if err != nil {
		return
	}

	return
}
