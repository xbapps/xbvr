package deo_remote

import (
	"encoding/binary"
	"encoding/json"
	"net"
	"time"

	"github.com/xbapps/xbvr/pkg/common"
)

type DeoPacket struct {
	Path          string  `json:"path,omitempty"`
	Duration      float64 `json:"duration,omitempty"`
	CurrentTime   float64 `json:"currentTime,omitempty"`
	PlaybackSpeed float64 `json:"playbackSpeed,omitempty"`
	PlayerState   int     `json:"playerState,omitempty"`
}

const PLAYING = 0
const PAUSED = 1

var DeoPlayerHost = ""

func DeoRemote() {
	for {
		deoLoop()
		time.Sleep(1 * time.Second)
	}
}

func deoLoop() error {
	if DeoPlayerHost == "" {
		return nil
	}
	conn, err := net.Dial("tcp", DeoPlayerHost+":23554")
	if err != nil {
		return err
	}

	common.Log.Info("Connected to DeoVR")

	for {
		// Read
		err := conn.SetReadDeadline(time.Now().Add(2 * time.Second))
		if err != nil {
			return err
		}

		// Check incoming packet length
		lenBuf := make([]byte, 4)
		_, err = conn.Read(lenBuf[:]) // recv data
		bodyLength := binary.LittleEndian.Uint32(lenBuf)

		// Read packet
		if bodyLength > 0 {
			recvBuf := make([]byte, bodyLength)
			_, err = conn.Read(recvBuf[:]) // recv data
			if err != nil {
				return err
			}

			packet := decodePacket(recvBuf)
			go TrackSession(packet)
		}

		// Write
		err = conn.SetWriteDeadline(time.Now().Add(1 * time.Second))
		if err != nil {
			return err
		}

		// Check if there's command queued, otherwise send ping packet
		packet := encodePacket(DeoPacket{})
		_, err = conn.Write(packet)
		if err != nil {
			return err
		}
	}
}

func encodePacket(packet DeoPacket) []byte {
	data, _ := json.Marshal(packet)
	header := make([]byte, 4)
	binary.LittleEndian.PutUint32(header, uint32(len(data)))

	return append(header, data...)
}

func decodePacket(data []byte) DeoPacket {
	var packet DeoPacket
	json.Unmarshal(data, &packet)
	return packet
}
