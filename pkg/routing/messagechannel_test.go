package routing_test

import (
	"sync"
	"testing"

	"github.com/underpin-korea/protocol/livekit"

	"github.com/underpin-korea/livekit_server_go/pkg/routing"
)

func TestMessageChannel_WriteMessageClosed(t *testing.T) {
	// ensure it doesn't panic when written to after closing
	m := routing.NewMessageChannel()
	go func() {
		for msg := range m.ReadChan() {
			if msg == nil {
				return
			}
		}
	}()

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			_ = m.WriteMessage(&livekit.RTCNodeMessage{})
		}
	}()
	_ = m.WriteMessage(&livekit.RTCNodeMessage{})
	m.Close()
	_ = m.WriteMessage(&livekit.RTCNodeMessage{})

	wg.Wait()
}
