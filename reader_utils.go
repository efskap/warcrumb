package warcrumb

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
)

func readWORD(file io.Reader) (uint16, error) {
	buf := make([]byte, 2)
	if _, err := file.Read(buf); err != nil {
		return 0, err
	}
	return binary.LittleEndian.Uint16(buf), nil
}
func readDWORD(file io.Reader) (uint32, error) {
	buf := make([]byte, 4)
	if _, err := file.Read(buf); err != nil {
		return 0, err
	}
	return binary.LittleEndian.Uint32(buf), nil
}
func expectDWORD(file io.Reader, expected uint32) error {
	actual, err := readDWORD(file)
	if err != nil {
		return err
	}
	if actual != expected {
		return UnexpectedValueError{
			actual:   actual,
			expected: expected,
			hex:      true,
		}
	}
	return nil
}
func expectWORD(file io.Reader, expected uint16) error {
	actual, err := readWORD(file)
	if err != nil {
		return err
	}
	if actual != expected {
		return UnexpectedValueError{
			actual:   actual,
			expected: expected,
			hex:      true,
		}
	}
	return nil
}
func expectByte(file io.Reader, expected byte) error {
	buf := make([]byte, 1)
	_, err := file.Read(buf)
	if err != nil {
		return err
	}
	actual := buf[0]
	if actual != expected {
		return UnexpectedValueError{
			actual:   actual,
			expected: expected,
			hex:      true,
		}
	}
	return nil
}

type UnexpectedValueError struct {
	actual   interface{}
	expected interface{}
	hex      bool
}

func (u UnexpectedValueError) Error() string {
	if u.hex {
		return fmt.Sprintf("unexpected value: 0x%x (expected 0x%x)", u.actual, u.expected)
	} else {
		return fmt.Sprintf("unexpected value: 0x%d (expected 0x%d)", u.actual, u.expected)
	}
}

func readCompressedBlock(file io.Reader, reforged bool) ([]byte, error) {
	var n uint32
	var err error
	if reforged {
		n, err = readDWORD(file)
	} else {
		n16, err2 := readWORD(file)
		n = uint32(n16)
		err = err2
	}
	if err != nil {
		return nil, fmt.Errorf("error reading size of compressed data block: %w", err)
	}

	var expectedDecompressedLength uint32
	if reforged {
		expectedDecompressedLength, err = readDWORD(file)
	} else {
		edl, err2 := readWORD(file)
		expectedDecompressedLength = uint32(edl)
		err = err2
	}
	if err != nil {
		return nil, fmt.Errorf("error reading size of decompressed data block: %w", err)
	}
	_, err = readDWORD(file)
	if err != nil {
		return nil, fmt.Errorf("error reading unknown field: %w", err)
	}

	// padding to 8K bytes seems to have been a LIE
	/*
		offset, _ := file.Seek(0, io.SeekCurrent)
		next8KBoundary := offset+ int64(n) + 0x2000
		next8KBoundary -= next8KBoundary % 0x2000
		paddingN := next8KBoundary - int64(n) - offset
		fmt.Printf("padding bytes: 0x%x ; from 0x%x to 0x%x \n", paddingN, offset, int64(n) + paddingN + offset)
	*/
	compressedData := make([]byte, int(n))
	//fmt.Printf("buffer total length: 0x%x\n", len(compressedData))
	if _, err := file.Read(compressedData); err != nil {
		return nil, fmt.Errorf("error reading compressed data: %w", err)
	}
	zr, err := zlib.NewReader(bytes.NewReader(compressedData))
	if err != nil {
		return nil, fmt.Errorf("error decompressing: %w", err)
	}
	inflateBuffer := make([]byte, expectedDecompressedLength)
	actuallyInflatedBytes, err := zr.Read(inflateBuffer)
	err = zr.Close()
	if actuallyInflatedBytes != int(expectedDecompressedLength) {
		return inflateBuffer, fmt.Errorf("actuallyInflatedBytes (%d) != expectedDecompressedLength (%d)", actuallyInflatedBytes, expectedDecompressedLength)
	}
	return inflateBuffer, err
}

func decodeString(encoded string) []byte {
	decoded := make([]byte, 0, len(encoded))
	var mask byte
	for i := 0; i < len(encoded); i++ {
		if i%8 == 0 {
			mask = encoded[i]
			continue
		}
		if (mask & (0x1 << (i % 8))) == 0 {
			decoded = append(decoded, encoded[i]-1)
		} else {
			decoded = append(decoded, encoded[i])
		}

	}
	return decoded
}

func readLittleEndianString(file io.Reader, nBytes int) (string, error) {
	buf := make([]byte, nBytes)
	if _, err := file.Read(buf); err != nil {
		return "", err
	}

	// https://github.com/golang/go/wiki/SliceTricks#reversing
	for left, right := 0, len(buf)-1; left < right; left, right = left+1, right-1 {
		buf[left], buf[right] = buf[right], buf[left]
	}
	return string(buf), nil
}

// readLengthAndThenString reads a non-null terminated string represented by its length and then the bytes
func readLengthAndThenString(buffer *bytes.Buffer) (string, error) {
	stringLength, err := buffer.ReadByte()
	if err != nil {
		return "", fmt.Errorf("error reading string length: %w", err)
	}
	stringBytes := make([]byte, stringLength)
	_, err = buffer.Read(stringBytes)
	if err != nil {
		return "", fmt.Errorf("error reading string: %w", err)
	}
	return string(stringBytes), nil
}
