package tritonhttp

import "time"

const (
	CONNECT_TIMEOUT time.Duration = 6 * time.Second
	SEND_TIMEOUT    time.Duration = 6 * time.Second
	RECV_TIMEOUT    time.Duration = 6 * time.Second
)
