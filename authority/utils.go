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
	"github.com/pritunl/pritunl-hsm/config"
	"github.com/pritunl/pritunl-hsm/errortypes"
	"github.com/pritunl/pritunl-hsm/utils"
	"github.com/pritunl/pritunl-hsm/yubikey"
	"golang.org/x/crypto/ssh"
	"gopkg.in/mgo.v2/bson"
	"hash/fnv"
	"time"
)

func Sign(hsmSerial string, sshReq *SshRequest) (
	certMarshaled []byte, err error) {

	if sshReq.Serial != hsmSerial {
		err = &errortypes.AuthenticationError{
			errors.Wrap(err, "authority: HSM serial mismatch"),
		}
		return
	}

	yubi := yubikey.GetKey(sshReq.Serial)
	if yubi == nil {
		err = &errortypes.NotFoundError{
			errors.New("authority: Failed to find hsm"),
		}
		return
	}

	cert, err := utils.UnmarshalSshCertificate(sshReq.Certificate)
	if err != nil {
		return
	}

	serialHash := fnv.New64a()
	serialHash.Write([]byte(bson.NewObjectId().Hex()))
	cert.Serial = serialHash.Sum64()

	maxCertExpire := config.Config.MaxCertificateExpire
	if maxCertExpire == 0 {
		maxCertExpire = config.DefaultMaxCertificateExpire
	}

	validAfter := uint64(time.Now().Add(-5 * time.Minute).Unix())
	validBefore := uint64(time.Now().Add(
		time.Duration(maxCertExpire) * time.Second).Unix())

	validBefore = utils.Min(validBefore, cert.ValidBefore)
	if validBefore <= validAfter {
		err = &errortypes.ParseError{
			errors.New(
				"authority: Certificate expire out of range, check clock"),
		}
		return
	}

	cert.ValidAfter = validAfter
	cert.ValidBefore = validBefore

	yubikey.LockKey(sshReq.Serial)

	slot, err := yubi.Authentication()
	if err != nil {
		return
	}

	signer, err := ssh.NewSignerFromSigner(slot)
	if err != nil {
		yubikey.UnlockKey(sshReq.Serial)
		return
	}

	err = cert.SignCert(rand.Reader, signer)
	if err != nil {
		yubikey.UnlockKey(sshReq.Serial)
		return
	}

	yubikey.UnlockKey(sshReq.Serial)

	certMarshaled, err = utils.MarshalSshCertificate(cert)
	if err != nil {
		return
	}

	return
}

func UnmarshalPayload(token, secret string, payloadJson []byte) (
	msgId string, sshReq *SshRequest, err error) {

	payload := &HsmPayload{}
	err = json.Unmarshal(payloadJson, payload)
	if err != nil {
		err = &errortypes.ParseError{
			errors.Wrap(err, "authority: Failed to unmarshal payload"),
		}
		return
	}

	if payload.Id == "" || payload.Token == "" || payload.Iv == nil ||
		payload.Data == nil {

		err = &errortypes.ParseError{
			errors.Wrap(err, "authority: Invalid payload"),
		}
		return
	}

	if subtle.ConstantTimeCompare([]byte(token),
		[]byte(payload.Token)) != 1 {

		err = &errortypes.AuthenticationError{
			errors.Wrap(err, "authority: Invalid token"),
		}
		return
	}

	cipData := payload.Data
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
	cipIv := payload.Iv

	if len(cipIv) != aes.BlockSize {
		err = &errortypes.ParseError{
			errors.Wrap(err, "authority: Invalid payload iv length"),
		}
		return
	}

	if len(cipData) == 0 || len(cipData)%16 != 0 {
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

	msgId = payload.Id
	sshReq = &SshRequest{}

	err = json.Unmarshal(cipData, sshReq)
	if err != nil {
		err = &errortypes.ParseError{
			errors.Wrap(err, "authority: Failed to unmarshal payload data"),
		}
		return
	}

	return
}

func MarshalPayload(id, token, secret string, cert []byte) (
	payload *HsmPayload, err error) {

	data := &SshResponse{
		Type:        "ssh_certificate",
		Certificate: cert,
	}

	cipData, err := json.Marshal(data)
	if err != nil {
		err = &errortypes.ParseError{
			errors.Wrap(err, "authority: Failed to marshal certificate"),
		}
		return
	}

	pad := 16 - len(cipData)%16
	for i := 0; i < pad; i++ {
		cipData = append(cipData, 0)
	}

	encKeyHash := sha256.New()
	encKeyHash.Write([]byte(secret))
	cipKey := encKeyHash.Sum(nil)

	cipIv, err := utils.RandBytes(aes.BlockSize)
	if err != nil {
		return
	}

	block, err := aes.NewCipher(cipKey)
	if err != nil {
		err = &errortypes.ParseError{
			errors.Wrap(err, "authority: Failed to load cipher"),
		}
		return
	}

	mode := cipher.NewCBCEncrypter(block, cipIv)
	mode.CryptBlocks(cipData, cipData)

	hashFunc := hmac.New(sha512.New, []byte(secret))
	hashFunc.Write(cipData)
	rawSignature := hashFunc.Sum(nil)
	sig := base64.StdEncoding.EncodeToString(rawSignature)

	payload = &HsmPayload{
		Id:        id,
		Token:     token,
		Iv:        cipIv,
		Signature: sig,
		Data:      cipData,
	}

	return
}
