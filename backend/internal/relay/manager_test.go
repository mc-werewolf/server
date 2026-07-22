package relay

import (
	"bytes"
	"testing"
)

func TestRelayFrameRoundTrip(t *testing.T) {
	frame, err := encodeFrame("203.0.113.10:54321", []byte{1, 2, 3, 4})
	if err != nil {
		t.Fatal(err)
	}
	address, payload, err := decodeFrame(frame)
	if err != nil {
		t.Fatal(err)
	}
	if address != "203.0.113.10:54321" || !bytes.Equal(payload, []byte{1, 2, 3, 4}) {
		t.Fatalf("unexpected decoded frame: %s %v", address, payload)
	}
}

func TestRejectsInvalidRelayFrames(t *testing.T) {
	for _, frame := range [][]byte{{}, {0}, {0, 5, 1}} {
		if _, _, err := decodeFrame(frame); err == nil {
			t.Fatalf("expected error for %v", frame)
		}
	}
}
