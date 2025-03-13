package tls

import (
	"bytes"
	"fmt"
	"io"
	"net"
)

// PeekSNI reads enough bytes from conn to extract the TLS SNI server name
// without decrypting. Returns a buffered conn so the peeked bytes are not lost.
func PeekSNI(conn net.Conn) (serverName string, buffered net.Conn, err error) {
	// TLS ClientHello is at most ~16KB; 4096 is enough for SNI.
	buf := make([]byte, 4096)
	n, err := conn.Read(buf)
	if err != nil && err != io.EOF {
		return "", nil, fmt.Errorf("peek SNI read: %w", err)
	}
	buf = buf[:n]
	serverName = extractSNI(buf)
	buffered = &bufConn{Conn: conn, r: bytes.NewReader(buf)}
	return serverName, buffered, nil
}

// extractSNI parses a TLS ClientHello record and returns the SNI extension value.
// Returns "" if not found or the record is malformed.
func extractSNI(data []byte) string {
	if len(data) < 5 {
		return ""
	}
	// Record layer: type(1) version(2) length(2)
	if data[0] != 0x16 { // Handshake
		return ""
	}
	if len(data) < 5+int(uint16(data[3])<<8|uint16(data[4])) {
		return ""
	}
	// Handshake: type(1) length(3)
	pos := 5
	if data[pos] != 0x01 { // ClientHello
		return ""
	}
	pos += 4
	// ClientHello: version(2) random(32) session_id_len(1) ...
	if pos+35 > len(data) {
		return ""
	}
	pos += 2 + 32 // skip version + random
	sessionLen := int(data[pos])
	pos += 1 + sessionLen
	if pos+2 > len(data) {
		return ""
	}
	cipherLen := int(uint16(data[pos])<<8 | uint16(data[pos+1]))
	pos += 2 + cipherLen
	if pos+1 > len(data) {
		return ""
	}
	compLen := int(data[pos])
	pos += 1 + compLen
	if pos+2 > len(data) {
		return ""
	}
	extTotal := int(uint16(data[pos])<<8 | uint16(data[pos+1]))
	pos += 2
	end := pos + extTotal
	for pos+4 <= end && pos+4 <= len(data) {
		extType := uint16(data[pos])<<8 | uint16(data[pos+1])
		extLen := int(uint16(data[pos+2])<<8 | uint16(data[pos+3]))
		pos += 4
		if extType == 0x0000 && pos+extLen <= len(data) { // SNI extension
			// list_len(2) type(1) name_len(2) name
			if extLen < 5 {
				return ""
			}
			nameLen := int(uint16(data[pos+3])<<8 | uint16(data[pos+4]))
			if pos+5+nameLen <= len(data) {
				return string(data[pos+5 : pos+5+nameLen])
			}
		}
		pos += extLen
	}
	return ""
}

// bufConn is a net.Conn that replays already-read bytes before forwarding to the real conn.
type bufConn struct {
	net.Conn
	r *bytes.Reader
}

func (c *bufConn) Read(b []byte) (int, error) {
	if c.r.Len() > 0 {
		return c.r.Read(b)
	}
	return c.Conn.Read(b)
}
