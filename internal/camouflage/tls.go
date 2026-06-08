// Package camouflage — ترافیک tunnel رو شبیه TLS 1.3 میکنه
// DPI فکر میکنه HTTPS هست
package camouflage

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"time"
)

const (
	// TLS record types
	tlsHandshake   = 0x16
	tlsAppData     = 0x17
	tlsVersion     = 0x0303 // TLS 1.2 wire format (TLS 1.3 هم همینو استفاده میکنه)
	tlsClientHello = 0x01
	tlsServerHello = 0x02

	maxTLSRecordSize = 16384 // 16KB — حد استاندارد TLS record
)

// TLSConn — یه conn که ترافیکش شبیه TLS هست
type TLSConn struct {
	conn      net.Conn
	isClient  bool
	handshake bool
}

// Wrap — یه connection معمولی رو به TLS camouflage تبدیل میکنه
func Wrap(conn net.Conn, isClient bool) *TLSConn {
	return &TLSConn{conn: conn, isClient: isClient}
}

// Handshake — یه TLS handshake جعلی انجام میده
func (c *TLSConn) Handshake() error {
	if c.handshake {
		return nil
	}
	if c.isClient {
		if err := c.sendClientHello(); err != nil {
			return err
		}
		if err := c.readServerHello(); err != nil {
			return err
		}
	} else {
		if err := c.readClientHello(); err != nil {
			return err
		}
		if err := c.sendServerHello(); err != nil {
			return err
		}
	}
	c.handshake = true
	return nil
}

// Write — داده رو با TLS Application Data record میفرسته
func (c *TLSConn) Write(data []byte) (int, error) {
	frame := makeTLSRecord(tlsAppData, data)
	_, err := c.conn.Write(frame)
	if err != nil {
		return 0, err
	}
	return len(data), nil
}

// Read — باگ ۴ fix: دقیقاً به اندازه dataLen میخونه؛ اگه buf کوچیکه error میده نه truncate
func (c *TLSConn) Read(buf []byte) (int, error) {
	// خوندن TLS record header (5 بایت ثابت)
	header := make([]byte, 5)
	if _, err := io.ReadFull(c.conn, header); err != nil {
		return 0, err
	}

	dataLen := int(binary.BigEndian.Uint16(header[3:5]))

	// اگه buf کوچیکتر از dataLen باشه، باید بخونیم و دور بندازیم تا stream sync بمونه
	if dataLen > len(buf) {
		// بخون و دور بنداز تا stream خراب نشه
		discard := make([]byte, dataLen)
		if _, err := io.ReadFull(c.conn, discard); err != nil {
			return 0, err
		}
		return 0, fmt.Errorf("camouflage: buffer too small (need %d, have %d)", dataLen, len(buf))
	}

	return io.ReadFull(c.conn, buf[:dataLen])
}

func (c *TLSConn) Close() error                       { return c.conn.Close() }
func (c *TLSConn) LocalAddr() net.Addr                { return c.conn.LocalAddr() }
func (c *TLSConn) RemoteAddr() net.Addr               { return c.conn.RemoteAddr() }
func (c *TLSConn) SetDeadline(t time.Time) error      { return c.conn.SetDeadline(t) }
func (c *TLSConn) SetReadDeadline(t time.Time) error  { return c.conn.SetReadDeadline(t) }
func (c *TLSConn) SetWriteDeadline(t time.Time) error { return c.conn.SetWriteDeadline(t) }

func (c *TLSConn) sendClientHello() error {
	random := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, random); err != nil {
		return err
	}
	cipherSuites := []byte{
		0x13, 0x01, // TLS_AES_128_GCM_SHA256
		0x13, 0x02, // TLS_AES_256_GCM_SHA384
		0x13, 0x03, // TLS_CHACHA20_POLY1305_SHA256
		0xc0, 0x2b, // TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256
		0xc0, 0x2c, // TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384
	}
	hello := buildClientHello(random, cipherSuites)
	frame := makeTLSRecord(tlsHandshake, hello)
	_, err := c.conn.Write(frame)
	return err
}

func (c *TLSConn) sendServerHello() error {
	random := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, random); err != nil {
		return err
	}
	hello := buildServerHello(random)
	frame := makeTLSRecord(tlsHandshake, hello)
	_, err := c.conn.Write(frame)
	return err
}

// readClientHello — باگ ۴ fix: header رو میخونه تا طول دقیق رو بدونه، بعد دقیقاً همون قدر میخونه
func (c *TLSConn) readClientHello() error {
	c.conn.SetReadDeadline(time.Now().Add(10 * time.Second))
	defer c.conn.SetReadDeadline(time.Time{})
	return c.drainTLSRecord()
}

// readServerHello — باگ ۴ fix: مثل readClientHello
func (c *TLSConn) readServerHello() error {
	c.conn.SetReadDeadline(time.Now().Add(10 * time.Second))
	defer c.conn.SetReadDeadline(time.Time{})
	return c.drainTLSRecord()
}

// drainTLSRecord — header رو میخونه، طول رو میگیره، دقیقاً همون قدر میخونه و دور میندازه
func (c *TLSConn) drainTLSRecord() error {
	header := make([]byte, 5)
	if _, err := io.ReadFull(c.conn, header); err != nil {
		return fmt.Errorf("camouflage: failed to read handshake header: %w", err)
	}
	msgLen := int(binary.BigEndian.Uint16(header[3:5]))
	if msgLen > maxTLSRecordSize {
		return fmt.Errorf("camouflage: handshake record too large (%d bytes)", msgLen)
	}
	body := make([]byte, msgLen)
	_, err := io.ReadFull(c.conn, body)
	return err
}

func makeTLSRecord(recordType byte, data []byte) []byte {
	record := make([]byte, 5+len(data))
	record[0] = recordType
	binary.BigEndian.PutUint16(record[1:3], tlsVersion)
	binary.BigEndian.PutUint16(record[3:5], uint16(len(data)))
	copy(record[5:], data)
	return record
}

func buildClientHello(random []byte, cipherSuites []byte) []byte {
	sessionID := make([]byte, 32)
	io.ReadFull(rand.Reader, sessionID)

	body := []byte{0x03, 0x03} // client version TLS 1.2
	body = append(body, random...)
	body = append(body, byte(len(sessionID)))
	body = append(body, sessionID...)
	body = append(body, byte(len(cipherSuites)>>8), byte(len(cipherSuites)))
	body = append(body, cipherSuites...)
	body = append(body, 0x01, 0x00) // compression methods

	hello := []byte{tlsClientHello}
	length := len(body)
	hello = append(hello, byte(length>>16), byte(length>>8), byte(length))
	hello = append(hello, body...)
	return hello
}

func buildServerHello(random []byte) []byte {
	body := []byte{0x03, 0x03}
	body = append(body, random...)
	body = append(body,
		0x00,       // session ID length
		0x13, 0x01, // TLS_AES_128_GCM_SHA256
		0x00,       // compression method
	)
	hello := []byte{tlsServerHello}
	length := len(body)
	hello = append(hello, byte(length>>16), byte(length>>8), byte(length))
	hello = append(hello, body...)
	return hello
}
