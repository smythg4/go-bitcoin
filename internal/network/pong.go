package network

type PongMessage struct {
	Nonce []byte
}

func (pm *PongMessage) Serialize() ([]byte, error) {
	return pm.Nonce, nil
}

func (pm PongMessage) Command() string {
	return "pong"
}

type PingMessage struct {
	Nonce []byte
}

func (pm *PingMessage) Serialize() ([]byte, error) {
	return pm.Nonce, nil
}

func (pm PingMessage) Command() string {
	return "ping"
}
