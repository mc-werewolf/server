package relay

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

const maxDatagramSize = 65535

var ErrNoPorts = errors.New("no relay ports available")

type Manager struct {
	firstPort int
	lastPort  int
	mu        sync.Mutex
	active    map[string]struct{}
}

func NewManager(firstPort, lastPort int) *Manager {
	return &Manager{firstPort: firstPort, lastPort: lastPort, active: make(map[string]struct{})}
}

func (m *Manager) Serve(serverID string, w http.ResponseWriter, r *http.Request, ready func(int) error) error {
	if !m.reserve(serverID) {
		return errors.New("relay already connected")
	}
	defer m.release(serverID)
	udp, port, err := m.listen()
	if err != nil {
		return err
	}
	defer udp.Close()
	if err := ready(port); err != nil {
		return err
	}
	connection, err := websocket.Accept(w, r, nil)
	if err != nil {
		return err
	}
	defer connection.Close(websocket.StatusNormalClosure, "relay closed")
	connection.SetReadLimit(maxDatagramSize + 512)
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()
	if err := wsjson.Write(ctx, connection, map[string]any{"type": "ready", "port": port}); err != nil {
		return err
	}
	errChannel := make(chan error, 2)
	go func() { errChannel <- forwardUDPToWebSocket(ctx, udp, connection) }()
	go func() { errChannel <- forwardWebSocketToUDP(ctx, udp, connection) }()
	return <-errChannel
}

func (m *Manager) reserve(serverID string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, exists := m.active[serverID]; exists {
		return false
	}
	m.active[serverID] = struct{}{}
	return true
}

func (m *Manager) release(serverID string) {
	m.mu.Lock()
	delete(m.active, serverID)
	m.mu.Unlock()
}

func (m *Manager) listen() (*net.UDPConn, int, error) {
	for port := m.firstPort; port <= m.lastPort; port++ {
		connection, err := net.ListenUDP("udp", &net.UDPAddr{Port: port})
		if err == nil {
			return connection, port, nil
		}
	}
	return nil, 0, ErrNoPorts
}

func forwardUDPToWebSocket(ctx context.Context, udp *net.UDPConn, ws *websocket.Conn) error {
	payload := make([]byte, maxDatagramSize)
	for {
		if err := udp.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
			return err
		}
		size, remote, err := udp.ReadFromUDP(payload)
		if timeout, ok := err.(net.Error); ok && timeout.Timeout() {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
				continue
			}
		}
		if err != nil {
			return err
		}
		frame, err := encodeFrame(remote.String(), payload[:size])
		if err != nil {
			return err
		}
		if err := ws.Write(ctx, websocket.MessageBinary, frame); err != nil {
			return err
		}
	}
}

func forwardWebSocketToUDP(ctx context.Context, udp *net.UDPConn, ws *websocket.Conn) error {
	for {
		messageType, frame, err := ws.Read(ctx)
		if err != nil {
			return err
		}
		if messageType != websocket.MessageBinary {
			continue
		}
		address, payload, err := decodeFrame(frame)
		if err != nil {
			return err
		}
		remote, err := net.ResolveUDPAddr("udp", address)
		if err != nil {
			return err
		}
		if _, err := udp.WriteToUDP(payload, remote); err != nil {
			return err
		}
	}
}

func encodeFrame(address string, payload []byte) ([]byte, error) {
	if len(address) > 65535 {
		return nil, errors.New("UDP address is too long")
	}
	frame := make([]byte, 2+len(address)+len(payload))
	binary.BigEndian.PutUint16(frame[:2], uint16(len(address)))
	copy(frame[2:], address)
	copy(frame[2+len(address):], payload)
	return frame, nil
}

func decodeFrame(frame []byte) (string, []byte, error) {
	if len(frame) < 2 {
		return "", nil, errors.New("relay frame is too short")
	}
	addressLength := int(binary.BigEndian.Uint16(frame[:2]))
	if addressLength == 0 || len(frame) < 2+addressLength {
		return "", nil, fmt.Errorf("invalid relay address length: %d", addressLength)
	}
	return string(frame[2 : 2+addressLength]), frame[2+addressLength:], nil
}
