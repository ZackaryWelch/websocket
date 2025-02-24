package websocket_test

import (
	"context"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/ZackaryWelch/websocket"
	"github.com/ZackaryWelch/websocket/internal/test/wstest"
	"github.com/stretchr/testify/assert"
)

func TestWasm(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	c, resp, err := websocket.Dial(ctx, os.Getenv("WS_ECHO_SERVER_URL"), &websocket.DialOptions{
		Subprotocols: []string{"echo"},
	})
	assert.NoError(t, err)
	defer c.Close(websocket.StatusInternalError, "")

	assert.Equal(t, "subprotocol", "echo", c.Subprotocol())
	assert.Equal(t, "response code", http.StatusSwitchingProtocols, resp.StatusCode)

	c.SetReadLimit(65536)
	for i := 0; i < 10; i++ {
		err = wstest.Echo(ctx, c, 65536)
		assert.NoError(t, err)
	}

	err = c.Close(websocket.StatusNormalClosure, "")
	assert.NoError(t, err)
}
