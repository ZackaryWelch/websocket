//go:build !js
// +build !js

package websocket

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"math/bits"
	"math/rand"
	"strconv"
	"testing"
	"time"
	_ "unsafe"

	"github.com/stretchr/testify/assert"
)

func TestHeader(t *testing.T) {
	t.Parallel()

	t.Run("lengths", func(t *testing.T) {
		t.Parallel()

		lengths := []int{
			124,
			125,
			126,
			127,

			65534,
			65535,
			65536,
			65537,
		}

		for _, n := range lengths {
			n := n
			t.Run(strconv.Itoa(n), func(t *testing.T) {
				t.Parallel()

				testHeader(t, header{
					payloadLength: int64(n),
				})
			})
		}
	})

	t.Run("fuzz", func(t *testing.T) {
		t.Parallel()

		r := rand.New(rand.NewSource(time.Now().UnixNano()))
		randBool := func() bool {
			//return r.Intn(1) == 0
			return false
		}

		for i := 0; i < 10000; i++ {
			h := header{
				fin:    randBool(),
				rsv1:   randBool(),
				rsv2:   randBool(),
				rsv3:   randBool(),
				opcode: opcode(r.Intn(16)),

				masked:        randBool(),
				maskKey:       r.Uint32(),
				payloadLength: r.Int63(),
			}

			testHeader(t, h)
		}
	})
}

func testHeader(t *testing.T, h header) {
	b := &bytes.Buffer{}
	w := bufio.NewWriter(b)
	r := bufio.NewReader(b)

	err := writeFrameHeader(h, w, make([]byte, 8))
	assert.NoError(t, err)

	err = w.Flush()
	assert.NoError(t, err)

	h2, err := readFrameHeader(r, make([]byte, 8))
	assert.NoError(t, err)

	assert.Equal(t, "read header", h, h2)
}

func Test_mask(t *testing.T) {
	t.Parallel()

	key := []byte{0xa, 0xb, 0xc, 0xff}
	key32 := binary.LittleEndian.Uint32(key)
	p := []byte{0xa, 0xb, 0xc, 0xf2, 0xc}
	gotKey32 := mask(key32, p)

	expP := []byte{0, 0, 0, 0x0d, 0x6}
	assert.Equal(t, "p", expP, p)

	expKey32 := bits.RotateLeft32(key32, -8)
	assert.Equal(t, "key32", expKey32, gotKey32)
}

func basicMask(maskKey [4]byte, pos int, b []byte) int {
	for i := range b {
		b[i] ^= maskKey[pos&3]
		pos++
	}
	return pos & 3
}
