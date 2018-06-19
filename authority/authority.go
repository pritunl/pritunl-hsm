package authority

type SshPermissions struct {
	CriticalOptions map[string]string `json:"critical_options"`
	Extensions      map[string]string `json:"extensions"`
}

type SshCertificate struct {
	Key             string         `json:"key"`
	CertType        uint32         `json:"cert_type"`
	KeyId           string         `json:"key_id"`
	ValidPrincipals []string       `json:"valid_principals"`
	Ttl             int            `json:"ttl"`
	Permissions     SshPermissions `json:"permissions"`
}

type SshCsr struct {
	Type        string         `json:"type"`
	Serial      string         `json:"serial"`
	Certificate SshCertificate `json:"certificate"`
}

type HsmPayload struct {
	Id        string `json:"id"`
	Token     string `json:"token"`
	Signature string `json:"signature"`
	Iv        string `json:"iv"`
	Data      string `json:"data"`
}
