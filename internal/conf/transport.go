package conf

import (
	"fmt"
	"slices"
)

type Transport struct {
	Protocol string `yaml:"protocol"`
	Conn     int    `yaml:"conn"`
	TCPBuf   int    `yaml:"tcpbuf"`
	UDPBuf   int    `yaml:"udpbuf"`
	KCP      *KCP   `yaml:"kcp"`
}

func (t *Transport) setDefaults(role string) {
	if t.Conn == 0 {
		t.Conn = 3
	}
	if t.TCPBuf == 0 {
		t.TCPBuf = 134217728 // 128 MB
	}
	if t.UDPBuf == 0 {
		t.UDPBuf = 134217728 // 128 MB
	}
	switch t.Protocol {
	case "kcp":
		// باگ ۶ fix: اگه بلوک kcp در config نباشه، KCP == nil هست و panic میکنه
		if t.KCP == nil {
			t.KCP = &KCP{}
		}
		t.KCP.setDefaults(role)
	}
}

func (t *Transport) validate() []error {
	var errors []error
	validProtocols := []string{"kcp"}
	if !slices.Contains(validProtocols, t.Protocol) {
		errors = append(errors, fmt.Errorf("transport protocol must be one of: %v", validProtocols))
	}
	if t.Conn < 1 || t.Conn > 256 {
		errors = append(errors, fmt.Errorf("conn must be between 1-256"))
	}
	switch t.Protocol {
	case "kcp":
		// باگ ۶ fix: nil guard مجدد در validate — setDefaults ممکنه صدا زده نشده باشه
		if t.KCP == nil {
			t.KCP = &KCP{}
		}
		errors = append(errors, t.KCP.validate()...)
	}
	return errors
}
