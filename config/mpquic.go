package config

import (
	"crypto/tls"
	"github.com/lucas-clemente/quic-go"
)

type MPQUICConfig struct {
	EnableMPTCP bool
	Paths      []PathConfig
}

type PathConfig struct {
	LocalAddr  string
	RemoteAddr string
	Priority   int
}

func NewMPQUICConfig() *MPQUICConfig {
	return &MPQUICConfig{
		EnableMPTCP: true,
		Paths: []PathConfig{
			{
				LocalAddr:  "0.0.0.0:4433",
				RemoteAddr: "localhost:4433",
				Priority:   1,
			},
			{
				LocalAddr:  "0.0.0.0:4434",
				RemoteAddr: "localhost:4434",
				Priority:   2,
			},
		},
	}
}

func GetTLSConfig() *tls.Config {
	return &tls.Config{
		InsecureSkipVerify: true,
		NextProtos:         []string{"mpquic-video"},
	}
}

func GetQuicConfig() *quic.Config {
	return &quic.Config{
		MaxIncomingStreams: 1000,
	}
} 