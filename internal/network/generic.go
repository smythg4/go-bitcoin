package network

type GenericMessage struct {
	command string
	payload []byte
}

func NewGenericMessage(command string, payload []byte) GenericMessage {
	return GenericMessage{
		command: command,
		payload: payload,
	}
}

func (g *GenericMessage) Serialize() ([]byte, error) {
	return g.payload, nil
}

func (g GenericMessage) Command() string {
	return g.command
}
