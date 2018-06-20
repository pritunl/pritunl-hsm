package authority

type SshRequest struct {
	Type        string `json:"type"`
	Serial      string `json:"serial"`
	Certificate []byte `json:"certificate"`
}

type SshResponse struct {
	Type        string `json:"type"`
	Certificate []byte `json:"certificate"`
}

type HsmPayload struct {
	Id        string `json:"id"`
	Token     string `json:"token"`
	Signature string `json:"signature"`
	Iv        []byte `json:"iv"`
	Data      []byte `json:"data"`
}
