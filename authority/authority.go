package authority

type SshRequest struct {
	Serial      string `json:"serial"`
	Certificate []byte `json:"certificate"`
}

type SshResponse struct {
	Certificate []byte `json:"certificate"`
}

type HsmStatus struct {
	Status       string `json:"status"`
	SshPublicKey string `json:"ssh_public_key"`
}

type HsmPayload struct {
	Id        string `json:"id"`
	Token     string `json:"token"`
	Signature string `json:"signature"`
	Iv        []byte `json:"iv"`
	Type      string `json:"type"`
	Data      []byte `json:"data"`
}
