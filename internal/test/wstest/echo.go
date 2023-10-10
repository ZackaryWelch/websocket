package wstest

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"reflect"
	"time"

	"github.com/ZackaryWelch/websocket"
	"github.com/ZackaryWelch/websocket/internal/test/xrand"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"google.golang.org/protobuf/proto"
)

// EchoLoop echos every msg received from c until an error
// occurs or the context expires.
// The read limit is set to 1 << 30.
func EchoLoop(ctx context.Context, c *websocket.Conn) error {
	defer c.Close(websocket.StatusInternalError, "")

	c.SetReadLimit(1 << 30)

	ctx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()

	b := make([]byte, 32<<10)
	for {
		typ, r, err := c.Reader(ctx)
		if err != nil {
			return err
		}

		w, err := c.Writer(ctx, typ)
		if err != nil {
			return err
		}

		_, err = io.CopyBuffer(w, r, b)
		if err != nil {
			return err
		}

		err = w.Close()
		if err != nil {
			return err
		}
	}
}

// Echo writes a message and ensures the same is sent back on c.
func Echo(ctx context.Context, c *websocket.Conn, max int) error {
	expType := websocket.MessageBinary
	if xrand.Bool() {
		expType = websocket.MessageText
	}

	msg := randMessage(expType, xrand.Int(max))

	writeErr := Go(func() error {
		return c.Write(ctx, expType, msg)
	})

	actType, act, err := c.Read(ctx)
	if err != nil {
		return err
	}

	err = <-writeErr
	if err != nil {
		return err
	}

	if expType != actType {
		return fmt.Errorf("unexpected message typ (%v): %v", expType, actType)
	}

	if !bytes.Equal(msg, act) {
		return fmt.Errorf("unexpected msg read: %v", cmp.Diff(msg, act, cmpopts.EquateErrors(), cmp.Exporter(func(r reflect.Type) bool { return true }), cmp.Comparer(proto.Equal)))
	}

	return nil
}

func randMessage(typ websocket.MessageType, n int) []byte {
	if typ == websocket.MessageBinary {
		return xrand.Bytes(n)
	}
	return []byte(xrand.String(n))
}

func Go(fn func() error) <-chan error {
	errs := make(chan error, 1)
	go func() {
		defer func() {
			r := recover()
			if r != nil {
				select {
				case errs <- fmt.Errorf("panic in go fn: %v", r):
				default:
				}
			}
		}()
		errs <- fn()
	}()

	return errs
}
