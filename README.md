# MPQUIC Video Streamer

A high-performance video streaming application leveraging Multipath QUIC (MPQUIC) protocol for efficient video transmission across multiple network paths.

## Features
- Real-time video streaming using MPQUIC
- Multi-path network support with configurable priorities
- TLS encryption with custom protocol support
- WebSocket-based streaming
- FFmpeg integration for video processing
- Configurable stream settings

## Prerequisites
- Go 1.17+
- FFmpeg
- quic-go v0.27.0
- gorilla/websocket

## Quick Start

1. **Install Dependencies**
```  bash
go mod download
```

3. **Run Server**
``` bash
go run server/main.go
```
4. **Access Web Interface**

## Configuration
Default MPQUIC configuration includes:
- Two network paths (ports 4433 and 4434)
- TLS with "mpquic-video" protocol
- Maximum 1000 incoming streams
- MPTCP enabled by default

Example configuration in `config/mpquic.go`:
``` go
func NewMPQUICConfig() MPQUICConfig {
return &MPQUICConfig{
EnableMPTCP: true,
Paths: []PathConfig{
{
LocalAddr: "0.0.0.0:4433",
RemoteAddr: "localhost:4433",
Priority: 1,
},
{
LocalAddr: "0.0.0.0:4434",
RemoteAddr: "localhost:4434",
Priority: 2,
},
},
}
}
```
## Project Structure
```
mpquic_streamer/
├── config/
│ └── mpquic.go # MPQUIC configuration
├── server/
│ └── main.go # Server implementation
├── camera/ # Camera handling
├── static/ # Web interface
└── go.mod
```

## Security Notes
- TLS encryption enabled
- Development mode uses InsecureSkipVerify
- Configure proper certificates for production use


## Contact
- GitHub: [@prabhastdvj](https://github.com/prabhastdvj)
- Project Link: [MPQUIC-VIDEO-STREAM](https://github.com/prabhastdvj/MPQUIC-VIDEO-STREAM)
