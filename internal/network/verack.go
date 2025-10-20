package network

type VerackMessage struct {
}

func (vm *VerackMessage) Serialize() ([]byte, error) {
	return []byte{}, nil
}

func (vm VerackMessage) Command() string {
	return "verack"
}
