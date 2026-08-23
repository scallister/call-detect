package tray

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/png"
)

const iconSize = 32

// IdleICO is a muted gray circle for the notification area.
func IdleICO() []byte { return StatusICO(false, false, false) }

// BusyICO is a green circle shown while a call is detected.
func BusyICO() []byte { return StatusICO(true, false, false) }

// ErrorICO is a red circle shown when the last webhook POST failed.
func ErrorICO() []byte { return StatusICO(false, true, false) }

// IdlePNG is the gray ring as a PNG (macOS / Linux).
func IdlePNG() []byte { return StatusPNG(false, false, false) }

// BusyPNG is the green ring as a PNG.
func BusyPNG() []byte { return StatusPNG(true, false, false) }

// ErrorPNG is the red ring as a PNG.
func ErrorPNG() []byte { return StatusPNG(false, true, false) }

// IdleARGB is the gray ring as StatusNotifier ARGB32 pixels.
func IdleARGB() []byte { return StatusARGB(false, false, false) }

// BusyARGB is the green ring as StatusNotifier ARGB32 pixels.
func BusyARGB() []byte { return StatusARGB(true, false, false) }

// ErrorARGB is the red ring as StatusNotifier ARGB32 pixels.
func ErrorARGB() []byte { return StatusARGB(false, true, false) }

// StatusICO is the notification-area icon for the current tray state.
func StatusICO(busy, failed, update bool) []byte {
	r, g, b := ringColor(busy, failed)
	return circleICO(r, g, b, update)
}

// StatusPNG is the PNG icon for the current tray state.
func StatusPNG(busy, failed, update bool) []byte {
	r, g, b := ringColor(busy, failed)
	return circlePNG(r, g, b, update)
}

// StatusARGB is the StatusNotifier pixmap for the current tray state.
func StatusARGB(busy, failed, update bool) []byte {
	r, g, b := ringColor(busy, failed)
	return circleARGB(r, g, b, update)
}

func ringColor(busy, failed bool) (byte, byte, byte) {
	switch {
	case failed:
		return 0xE0, 0x3C, 0x31
	case busy:
		return 0x2E, 0xCC, 0x40
	default:
		return 0x88, 0x88, 0x88
	}
}

func withUpMark(img *image.RGBA) {
	const cr, cg, cb byte = 0xF4, 0xF4, 0xF4
	cx, top, h, halfW := 15.5, 11.0, 9.0, 6.5
	for y := 0; y < iconSize; y++ {
		for x := 0; x < iconSize; x++ {
			relY := float64(y) - top
			if relY < 0 || relY > h {
				continue
			}
			half := (relY / h) * halfW
			dx := float64(x) - cx
			if dx < 0 {
				dx = -dx
			}
			if dx <= half {
				i := img.PixOffset(x, y)
				img.Pix[i+0] = cr
				img.Pix[i+1] = cg
				img.Pix[i+2] = cb
				img.Pix[i+3] = 0xFF
			}
		}
	}
}

func ringRGBA(cr, cg, cb byte) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, iconSize, iconSize))
	cx, cy := float64(iconSize)/2-0.5, float64(iconSize)/2-0.5
	outer := float64(iconSize)/2 - 1.5
	inner := outer - 3
	for y := 0; y < iconSize; y++ {
		for x := 0; x < iconSize; x++ {
			dx := float64(x) - cx
			dy := float64(y) - cy
			d2 := dx*dx + dy*dy
			if d2 <= outer*outer && d2 >= inner*inner {
				i := img.PixOffset(x, y)
				img.Pix[i+0] = cr
				img.Pix[i+1] = cg
				img.Pix[i+2] = cb
				img.Pix[i+3] = 0xFF
			}
		}
	}
	return img
}

func circlePNG(r, g, b byte, update bool) []byte {
	img := ringRGBA(r, g, b)
	if update {
		withUpMark(img)
	}
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}

// circleARGB is a StatusNotifier IconPixmap payload (ARGB, network byte order).
func circleARGB(cr, cg, cb byte, update bool) []byte {
	img := ringRGBA(cr, cg, cb)
	if update {
		withUpMark(img)
	}
	out := make([]byte, iconSize*iconSize*4)
	for i := 0; i < iconSize*iconSize; i++ {
		r, g, b, a := img.Pix[i*4], img.Pix[i*4+1], img.Pix[i*4+2], img.Pix[i*4+3]
		out[i*4+0] = a
		out[i*4+1] = r
		out[i*4+2] = g
		out[i*4+3] = b
	}
	return out
}

func circleICO(r, g, b byte, update bool) []byte {
	img := ringRGBA(r, g, b)
	if update {
		withUpMark(img)
	}
	xor := make([]byte, iconSize*iconSize*4)
	for y := 0; y < iconSize; y++ {
		for x := 0; x < iconSize; x++ {
			si := img.PixOffset(x, y)
			row := iconSize - 1 - y
			di := (row*iconSize + x) * 4
			xor[di+0] = img.Pix[si+2]
			xor[di+1] = img.Pix[si+1]
			xor[di+2] = img.Pix[si+0]
			xor[di+3] = img.Pix[si+3]
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
