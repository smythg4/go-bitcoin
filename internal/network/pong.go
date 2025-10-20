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
