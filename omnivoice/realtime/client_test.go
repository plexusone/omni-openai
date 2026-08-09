package realtime

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// TestReadLoop_NoSilentDropUnderBackpressure verifies that readLoop never
// silently drops server events when the consumer is slower than the socket:
// a full events channel must pause the read (backpressure) rather than
// discard audio/transcript events out from under the consumer.
func TestReadLoop_NoSilentDropUnderBackpressure(t *testing.T) {
	const eventCount = 250 // more than the events channel buffer (100)

	upgrader := websocket.Upgrader{}
	serverDone := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("upgrade: %v", err)
			return
		}
		defer conn.Close()

		// Blast events faster than any consumer could drain them; the
		// item_id carries the sequence number so order can be verified.
		for i := 0; i < eventCount; i++ {
			msg := fmt.Sprintf(`{"type":"response.audio.delta","item_id":"item_%d","delta":"AA=="}`, i)
			if err := conn.WriteMessage(websocket.TextMessage, []byte(msg)); err != nil {
				t.Errorf("write %d: %v", i, err)
				return
			}
		}
		// Orderly close so the client read loop terminates.
		deadline := time.Now().Add(time.Second)
		_ = conn.WriteControl(websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""), deadline)
		close(serverDone)
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	conn, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	if resp != nil && resp.Body != nil {
		resp.Body.Close()
	}

	session := &Session{
		conn:     conn,
		eventsCh: make(chan ServerEvent, 100),
		sendCh:   make(chan any, 100),
		closeCh:  make(chan struct{}),
	}
	session.wg.Add(1)
	go session.readLoop()

	// Wait until the server has written everything before consuming, so the
	// events channel is guaranteed to hit its capacity limit first.
	<-serverDone

	var mu sync.Mutex
	received := 0
	nextSeq := 0
	timeout := time.After(10 * time.Second)

consume:
	for {
		select {
		case event := <-session.eventsCh:
			switch e := event.(type) {
			case *ResponseAudioDeltaEvent:
				mu.Lock()
				want := fmt.Sprintf("item_%d", nextSeq)
				if e.ItemID != want {
					mu.Unlock()
					t.Fatalf("event out of order or dropped: got %q, want %q", e.ItemID, want)
				}
				nextSeq++
				received++
				done := received == eventCount
				mu.Unlock()
				if done {
					break consume
				}
			case *ErrorEvent:
				t.Fatalf("premature error event after %d/%d deltas: %s", received, eventCount, e.Error.Message)
			}
		case <-timeout:
			t.Fatalf("timed out with %d/%d events received — events were dropped", received, eventCount)
		}
	}

	close(session.closeCh)
	conn.Close()
	session.wg.Wait()

	if received != eventCount {
		t.Fatalf("received %d events, want %d", received, eventCount)
	}
}
