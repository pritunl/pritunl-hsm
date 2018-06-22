package socket

import (
	"time"
)

const (
	writeTimeout   = 10 * time.Second
	statusInterval = 30 * time.Second
	pingInterval   = 30 * time.Second
	pingWait       = 40 * time.Second
)
