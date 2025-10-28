package network

import (
	"errors"
	"fmt"
	"net"
	"sync"
	"time"
)

type MessageHandler func(NetworkEnvelope)

type SimpleNode struct {
	Addr    NetAddr
	conn    net.Conn
	TestNet bool
	Logging bool

	incoming chan NetworkEnvelope
	outgoing chan Message
	done     chan struct{}
	wg       sync.WaitGroup

	handlers map[string]MessageHandler

	// dedicated channels for messages we need to wait on
	channelsMap map[string]chan NetworkEnvelope
}

func NewSimpleNode(host string, port int, testNet, logging bool) (*SimpleNode, error) {
	ip := net.ParseIP(host)
	if ip == nil {
		return nil, fmt.Errorf("invalid ip address: %s", host)
	}
	ip16 := ip.To16()
	var address [16]byte
	copy(address[:], ip16)
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", host, port), 5*time.Second)
	if err != nil {
		return nil, fmt.Errorf("error connecting to %s:%d - %w", host, port, err)
	}
	sn := &SimpleNode{
		Addr: NetAddr{
			Services: 0,
			Address:  address,
			Port:     uint16(port),
		},
		conn:     conn,
		TestNet:  testNet,
		Logging:  logging,
		incoming: make(chan NetworkEnvelope, 10),
		outgoing: make(chan Message, 10),
		done:     make(chan struct{}),
		handlers: make(map[string]MessageHandler),

		// dedicated channels for message types (buffered to prevent drops)
		channelsMap: make(map[string]chan NetworkEnvelope),
	}

	sn.RegisterChannel("version", 1)
	sn.RegisterChannel("verack", 1)
	sn.RegisterChannel("headers", 1)
	sn.RegisterChannel("block", 1)
	sn.RegisterChannel("merkleblock", 1)
	sn.RegisterChannel("tx", 25)
	sn.RegisterChannel("cmpctblock", 1)
	sn.RegisterChannel("getblocktxn", 1)
	sn.RegisterChannel("blocktxn", 1)
	sn.RegisterChannel("sendcmpct", 1)
	sn.wg.Add(3)

	go sn.readLoop()
	go sn.sendLoop()
	go sn.messageLoop()

	// Auto-respond to ping messages
	sn.OnMessage("ping", func(env NetworkEnvelope) {
		if sn.Logging {
			fmt.Println("Auto-responding to ping")
		}
		pong := &PongMessage{Nonce: env.Payload}
		sn.Send(pong)
	})

	// Log received verack (no response needed)
	sn.OnMessage("verack", func(env NetworkEnvelope) {
		if sn.Logging {
			fmt.Println("Received verack")
		}
	})

	// Log protocol messages we don't care about (optional)
	sn.OnMessage("sendheaders", func(env NetworkEnvelope) {
		if sn.Logging {
			fmt.Println("Peer requested sendheaders (BIP 130)")
		}
	})

	sn.OnMessage("sendcmpct", func(env NetworkEnvelope) {
		if sn.Logging {
			fmt.Println("Peer requested compact blocks (BIP 152)")
		}
	})

	sn.OnMessage("feefilter", func(env NetworkEnvelope) {
		if sn.Logging {
			fmt.Println("Peer sent fee filter (BIP 133)")
		}
	})

	sn.OnMessage("inv", func(env NetworkEnvelope) {
		if sn.Logging {
			fmt.Println("Peer sent inv")
		}
	})

	return sn, nil
}

func (sn *SimpleNode) RegisterChannel(name string, bufSize int) {
	sn.channelsMap[name] = make(chan NetworkEnvelope, bufSize)
}

func (sn *SimpleNode) readLoop() {
	defer sn.wg.Done()
	defer close(sn.incoming) // reader is done

	for {
		select {
		case <-sn.done:
			return
		default:
			env, err := ParseNetworkEnvelope(sn.conn)
			if err != nil {
				if sn.Logging {
					fmt.Printf("read error: %v\n", err)
				}
				return
			}
			if sn.Logging {
				fmt.Printf("receiving: %s\n", env.Command)
			}

			select {
			case sn.incoming <- env:
			case <-sn.done:
				return
			}
		}
	}
}

func (sn *SimpleNode) sendLoop() {
	defer sn.wg.Done()

	for {
		select {
		case msg := <-sn.outgoing:
			// serialize and write to conn
			payload, err := msg.Serialize()
			if err != nil {
				if sn.Logging {
					fmt.Printf("serialization error: %v\n", err)
				}
				return
			}
			envelope, err := NewNetworkEnvelope(msg.Command(), payload, sn.TestNet)
			if err != nil {
				if sn.Logging {
					fmt.Printf("network envelope error: %v\n", err)
				}
				return
			}
			if sn.Logging {
				fmt.Printf("sending: %s\n", envelope)
			}
			data, err := envelope.Serialize()
			if err != nil {
				if sn.Logging {
					fmt.Printf("serialization error: %v\n", err)
				}
				return
			}
			_, err = sn.conn.Write(data)
			if err != nil {
				if sn.Logging {
					fmt.Printf("write error: %v\n", err)
				}
				return
			}
		case <-sn.done:
			return
		}
	}
}

func (sn *SimpleNode) Send(msg Message) error {
	// send a message to the connected node
	select {
	case sn.outgoing <- msg:
		return nil
	case <-sn.done:
		return fmt.Errorf("connection closed")
	}
}

func (sn *SimpleNode) messageLoop() {
	defer func() {
		sn.wg.Done()
		for _, ch := range sn.channelsMap {
			close(ch)
		}
	}()
	for env := range sn.incoming {
		// fan out to dedicated channels
		if ch, ok := sn.channelsMap[env.Command]; ok {
			// avoid blocking
			select {
			case ch <- env:
				// message sent successfully
			default:
				// channel full - drop message or log
				if sn.Logging {
					fmt.Printf("Warning: channel full for %s, dropping message\n", env.Command)
				}
			}
		}

		// also run handlers
		if handler, ok := sn.handlers[env.Command]; ok {
			go handler(env)
		}
	}
}

func (sn *SimpleNode) OnMessage(command string, handler MessageHandler) {
	sn.handlers[command] = handler
}

func (sn *SimpleNode) Handshake() error {
	msg := DefaultVersionMessage(net.IP(sn.Addr.Address[:]), sn.Addr.Port)
	if sn.Logging {
		fmt.Printf("📤 Sending version message with Services: %d\n", msg.Services)
	}
	err := sn.Send(&msg)
	if err != nil {
		return err
	}

	// blocks on receiving a version and verack response
	<-sn.channelsMap["version"]
	<-sn.channelsMap["verack"]

	// this explicit send solves the previous race condition
	// consider removing the automated verack response
	if err := sn.Send(&VerackMessage{}); err != nil {
		return err
	}

	if sn.Logging {
		fmt.Println("✓ Handshake complete!")
	}

	return nil
}

// default timeout of 5 seconds for a receive
func (sn *SimpleNode) Receive(command string) (NetworkEnvelope, error) {
	timeout := time.NewTimer(5 * time.Second)
	defer timeout.Stop()
	var ch chan NetworkEnvelope
	var ok bool
	if ch, ok = sn.channelsMap[command]; !ok {
		return NetworkEnvelope{}, errors.New("unknown command")
	}
	select {
	case env, ok := <-ch:
		if !ok {
			return NetworkEnvelope{}, errors.New("connection closed")
		}
		return env, nil
	case <-timeout.C:
		return NetworkEnvelope{}, fmt.Errorf("timeout waiting for %s", command)
	case <-sn.done:
		return NetworkEnvelope{}, errors.New("connection closed")
	}
}

// user configurable timeout parameter
func (sn *SimpleNode) ReceiveWithTimeout(command string, timeout time.Duration) (NetworkEnvelope, error) {
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	var ch chan NetworkEnvelope
	var ok bool
	if ch, ok = sn.channelsMap[command]; !ok {
		return NetworkEnvelope{}, errors.New("unknown command")
	}
	select {
	case env, ok := <-ch:
		if !ok {
			return NetworkEnvelope{}, errors.New("connection closed")
		}
		return env, nil
	case <-timer.C:
		return NetworkEnvelope{}, fmt.Errorf("timeout waiting for %s", command)
	case <-sn.done:
		return NetworkEnvelope{}, errors.New("connection closed")
	}
}

func (sn *SimpleNode) RequestHeaders(prevHash [32]byte) error {
	// TODO
	return nil
}

func (sn *SimpleNode) RequestMerkleBlock(blockHash [32]byte) error {
	// TODO
	return nil
}

func (sn *SimpleNode) Close() error {
	close(sn.done)
	sn.wg.Wait()
	err := sn.conn.Close()

	if sn.Logging {
		fmt.Printf("closing connection to %s...\n", sn.conn.RemoteAddr().String())
	}
	return err
}
