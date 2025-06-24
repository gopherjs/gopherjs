package sourcemapx

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"fmt"
	"go/token"
	"io"
	"reflect"
)

// A magic byte in the generated code output that indicates a beginning of a
// source map hint. The character has been chosen because it should never show
// up in valid generated code unescaped, other than for source map hint purposes.
const HintMagic byte = '\b'

// Hint is a container for a sourcemap hint that can be embedded into the
// generated code stream. Payload size and semantics depend on the nature of the
// hint.
//
// Within the stream, the hint is encoded in the following binary format:
//   - magic: 0x08 - ASCII backspace, magic symbol indicating the beginning of the hint;
//   - size: 16 bit, big endian unsigned int - the size of the payload.
//   - payload: [size]byte - the payload of the hint.
type Hint struct {
	Payload []byte
}

// FindHint returns the lowest index in the byte slice where a source map Hint
// is embedded or -1 if it isn't found. Invariant: if FindHint(b) != -1 then
// b[FindHint(b)] == '\b'.
func FindHint(b []byte) int {
	return bytes.IndexByte(b, HintMagic)
}

// ReadHint reads the Hint from the beginning of the byte slice and returns
// the hint and the number of bytes in the slice it occupies. The caller is
// expected to find the location of the hint using FindHint prior to calling
// this function.
//
// Returned hint payload does not share backing array with b.
//
// Function panics if:
//   - b[0] != '\b'
//   - len(b) < size + 3
func ReadHint(b []byte) (h Hint, length int) {
	if len(b) < 3 {
		panic(fmt.Errorf("byte slice too short to contain hint header: len(b) = %d", len(b)))
	}
	if b[0] != HintMagic {
		panic(fmt.Errorf("byte slice doesn't start with magic 0x%x: b[0] = 0x%x", HintMagic, b[0]))
	}
	size := int(binary.BigEndian.Uint16(b[1:3]))
	if len(b) < size+3 {
		panic(fmt.Errorf("byte slice it too short to contain hint payload: len(b) = %d, expected hint size: %d", len(b), size+3))
	}

	h.Payload = make([]byte, size)
	copy(h.Payload, b[3:])
	return h, size + 3
}

// WriteTo the encoded hint into the output stream. Panics if payload is longer
// than 0xFFFF bytes.
func (h *Hint) WriteTo(w io.Writer) (int64, error) {
	if len(h.Payload) > 0xFFFF {
		panic(fmt.Errorf("hint payload may not be longer than %d bytes, got: %d", 0xFFFF, len(h.Payload)))
	}
	encoded := []byte{HintMagic}
	encoded = binary.BigEndian.AppendUint16(encoded, uint16(len(h.Payload)))
	encoded = append(encoded, h.Payload...)

	n, err := w.Write(encoded)
	if err != nil {
		return int64(n), fmt.Errorf("failed to write hint: %w", err)
	}

	return int64(n), nil
}

// Pack the given value into hint's payload.
//
// Supported types: go/token.Pos.
//
// The first byte of the payload will indicate the encoded type, and the rest
// is an opaque, type-dependent binary representation of the type.
func (h *Hint) Pack(value any) error {
	payload := &bytes.Buffer{}
	// Write type flag.
	switch value.(type) {
	case token.Pos:
		payload.WriteByte(1)
	case Identifier:
		payload.WriteByte(2)
	default:
		return fmt.Errorf("unsupported hint payload type %T", value)
	}

	if err := gob.NewEncoder(payload).Encode(value); err != nil {
		return fmt.Errorf("failed to encode hint payload: %w", err)
	}

	h.Payload = payload.Bytes()
	return nil
}

// Unpack and return hint's payload, previously packed by Pack().
func (h *Hint) Unpack() (any, error) {
	if len(h.Payload) < 1 {
		return nil, fmt.Errorf("payload is too short to contain type flag")
	}
	var value any
	switch h.Payload[0] {
	case 1:
		v := token.NoPos
		value = &v
	case 2:
		value = &Identifier{}
	default:
		return nil, fmt.Errorf("unsupported hint payload type flag: %d", h.Payload[0])
	}
	if err := gob.NewDecoder(bytes.NewReader(h.Payload[1:])).Decode(value); err != nil {
		return nil, fmt.Errorf("failed to decode hint payload as %T: %w", value, err)
	}
	return reflect.ValueOf(value).Elem().Interface(), nil
}
