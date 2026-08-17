package tray

import (
	"bytes"
	"encoding/binary"
)

const iconSize = 32

// IdleICO is a muted gray circle for the notification area.
func IdleICO() []byte {
	return circleICO(0x88, 0x88, 0x88)
}

// BusyICO is a green circle shown while a call is detected.
func BusyICO() []byte {
	return circleICO(0x2E, 0xCC, 0x40)
}

func circleICO(r, g, b byte) []byte {
	xor := make([]byte, iconSize*iconSize*4)
	cx, cy := float64(iconSize)/2-0.5, float64(iconSize)/2-0.5
	outer := float64(iconSize)/2 - 1.5
	inner := outer - 3
	for y := 0; y < iconSize; y++ {
		for x := 0; x < iconSize; x++ {
			dx := float64(x) - cx
			dy := float64(y) - cy
			d2 := dx*dx + dy*dy
			// DIB is bottom-up
			row := iconSize - 1 - y
			i := (row*iconSize + x) * 4
			if d2 <= outer*outer && d2 >= inner*inner {
				xor[i+0] = b
				xor[i+1] = g
				xor[i+2] = r
				xor[i+3] = 0xFF
			}
		}
	}

	andRow := ((iconSize + 31) / 32) * 4
	and := make([]byte, andRow*iconSize)

	bih := make([]byte, 40)
	binary.LittleEndian.PutUint32(bih[0:4], 40)
	binary.LittleEndian.PutUint32(bih[4:8], uint32(iconSize))
	binary.LittleEndian.PutUint32(bih[8:12], uint32(iconSize*2))
	binary.LittleEndian.PutUint16(bih[12:14], 1)
	binary.LittleEndian.PutUint16(bih[14:16], 32)
	binary.LittleEndian.PutUint32(bih[20:24], uint32(len(xor)+len(and)))

	image := append(append(bih, xor...), and...)

	var buf bytes.Buffer
	_ = binary.Write(&buf, binary.LittleEndian, uint16(0))
	_ = binary.Write(&buf, binary.LittleEndian, uint16(1))
	_ = binary.Write(&buf, binary.LittleEndian, uint16(1))
	buf.WriteByte(byte(iconSize))
	buf.WriteByte(byte(iconSize))
	buf.WriteByte(0)
	buf.WriteByte(0)
	_ = binary.Write(&buf, binary.LittleEndian, uint16(1))
	_ = binary.Write(&buf, binary.LittleEndian, uint16(32))
	_ = binary.Write(&buf, binary.LittleEndian, uint32(len(image)))
	_ = binary.Write(&buf, binary.LittleEndian, uint32(6+16))
	buf.Write(image)
	return buf.Bytes()
}
