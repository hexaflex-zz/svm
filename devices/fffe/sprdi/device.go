// Package sprdi implements the Sprite Display Mk I
package sprdi

import (
	"fmt"
	"log"
	"unsafe"

	"github.com/go-gl/gl/v4.2-core/gl"
	"github.com/pkg/errors"

	"github.com/hexaflex/svm/devices"
	"github.com/hexaflex/svm/devices/fffe/cpu"
)

// Known interrupt operations.
const (
	setBackgroundPalette = iota
	setForegroundPalette
	setBackgroundSprites
	setForegroundSprites
	drawBackground
	drawForeground
	clearBackground
	clearForeground
	swap
)

// Various display and sprite properties.
const (
	DisplayWidth           = 256                // Display width in pixels.
	DisplayHeight          = 240                // Display height in pixels.
	PaletteSize            = 16                 // PaletteSize defines the number of colors in a color palette.
	BufferSize             = 256                // BufferSize defines the number of sprites stored in each internal sprite buffer.
	SpritePixelSize        = 8                  // SpritePixelSize defines the width and height in pixels, for a single sprite.
	SpriteByteSize         = 32                 // SpriteByteSize defines the size of a sprite in bytes in CPU memory: 4bpp.
	internalSpriteByteSize = SpriteByteSize * 2 // Size of sprite data in internal device memory: 8bpp.
)

// The number of tiles in each background dimension.
const (
	HorizontalTileCount = DisplayWidth / SpritePixelSize
	VerticalTileCount   = DisplayHeight / SpritePixelSize
)

// Device defines all internal doodads for the display.
type Device struct {
	palette         [PaletteSize * 2 * 4]float32
	sprites         [BufferSize * 2 * SpritePixelSize * SpritePixelSize]byte
	background      [DisplayWidth * DisplayHeight]byte
	foreground      [DisplayWidth * DisplayHeight]byte
	empty           [DisplayWidth * DisplayHeight]byte
	shader          uint32
	vao             uint32
	vbo             uint32
	backgroundTex   uint32
	foregroundTex   uint32
	paletteDirty    bool
	backgroundDirty bool
	foregroundDirty bool
	initialized     bool
}

var _ devices.Device = &Device{}

// New creates a new device.
func New() *Device {
	return &Device{}
}

// Draw renders the display contents.
func (d *Device) Draw() {
	if !d.initialized {
		return
	}

	gl.UseProgram(d.shader)
	gl.BindVertexArray(d.vao)

	gl.ActiveTexture(gl.TEXTURE0)
	gl.BindTexture(gl.TEXTURE_2D, d.backgroundTex)

	gl.ActiveTexture(gl.TEXTURE1)
	gl.BindTexture(gl.TEXTURE_2D, d.foregroundTex)

	gl.DrawArrays(gl.TRIANGLES, 0, 6)
}

// ID returns the device identifier.
func (d *Device) ID() devices.ID {
	return devices.NewID(0xfffe, 0x0002)
}

// Startup initializes device resources.
func (d *Device) Startup(devices.IntFunc) error {
	var err error

	d.shader, err = compileProgram(vertex, "", fragment)
	if err != nil {
		log.Fatal(err)
		return errors.Wrapf(err, "failed to compile shaders")
	}

	gl.UseProgram(d.shader)

	gl.GenVertexArrays(1, &d.vao)
	gl.BindVertexArray(d.vao)

	gl.GenBuffers(1, &d.vbo)
	gl.BindBuffer(gl.ARRAY_BUFFER, d.vbo)
	gl.BufferData(gl.ARRAY_BUFFER, len(quadVertices)*4, gl.Ptr(quadVertices), gl.STATIC_DRAW)

	vertAttrib := uint32(gl.GetAttribLocation(d.shader, glStr("vertPos")))
	texCoordAttrib := uint32(gl.GetAttribLocation(d.shader, glStr("vertTexCoord")))

	gl.EnableVertexAttribArray(vertAttrib)
	gl.VertexAttribPointer(vertAttrib, 3, gl.FLOAT, false, 5*4, gl.PtrOffset(0))

	gl.EnableVertexAttribArray(texCoordAttrib)
	gl.VertexAttribPointer(texCoordAttrib, 2, gl.FLOAT, false, 5*4, gl.PtrOffset(3*4))

	d.backgroundTex = makeTexture()
	d.foregroundTex = makeTexture()

	d.paletteDirty = true
	d.backgroundDirty = true
	d.foregroundDirty = true
	d.initialized = true
	d.swap()
	return nil
}

// Shutdown clears up device resources.
func (d *Device) Shutdown() error {
	d.initialized = false
	gl.DeleteTextures(1, &d.foregroundTex)
	gl.DeleteTextures(1, &d.backgroundTex)
	gl.DeleteBuffers(1, &d.vbo)
	gl.DeleteVertexArrays(1, &d.vao)
	gl.DeleteProgram(d.shader)
	return nil
}

// Int triggers an interrupt on the device. The device can read from- and write to system memory.
func (d *Device) Int(mem devices.Memory) {
	switch mem.U16(cpu.R0) {
	case setBackgroundPalette:
		d.setBackgroundPalette(mem)
	case setForegroundPalette:
		d.setForegroundPalette(mem)
	case setBackgroundSprites:
		d.setBackgroundSprites(mem)
	case setForegroundSprites:
		d.setForegroundSprites(mem)
	case drawBackground:
		d.drawBackground(mem)
	case drawForeground:
		d.drawForeground(mem)
	case clearBackground:
		d.clearBackground()
	case clearForeground:
		d.clearForeground()
	case swap:
		d.swap()
	}
}

func (d *Device) swap() {
	if d.paletteDirty {
		gl.UseProgram(d.shader)
		palette := gl.GetUniformLocation(d.shader, glStr("palette"))
		gl.Uniform4fv(palette, 32, &d.palette[0])
		d.paletteDirty = false
	}

	if d.backgroundDirty {
		uploadTexture(d.backgroundTex, gl.RED, DisplayWidth, DisplayHeight, gl.RED, gl.UNSIGNED_BYTE, d.background[:])
		d.backgroundDirty = false
	}

	if d.foregroundDirty {
		uploadTexture(d.foregroundTex, gl.RED, DisplayWidth, DisplayHeight, gl.RED, gl.UNSIGNED_BYTE, d.foreground[:])
		d.foregroundDirty = false
	}
}

func (d *Device) drawBackground(mem devices.Memory) {
	srcAddr := mem.U16(cpu.R1)
	dstAddr := mem.U16(cpu.R2)
	count := mem.U16(cpu.R3)
	bg := d.background[:]

	dx := dstAddr % HorizontalTileCount
	dy := dstAddr / HorizontalTileCount
	dstAddr = dy*DisplayWidth*SpritePixelSize + dx*SpritePixelSize

	for i := 0; i < count; i++ {
		d.drawSprite(bg, dstAddr, mem.U8(srcAddr), true)
		dstAddr += 8
		srcAddr += 1
	}

	d.backgroundDirty = true
}

func (d *Device) drawForeground(mem devices.Memory) {
	srcAddr := mem.U16(cpu.R1)
	count := mem.U16(cpu.R2)
	fg := d.foreground[:]

	for i := 0; i < count; i++ {
		s := mem.U8(srcAddr + 0)
		x := mem.U8(srcAddr + 1)
		y := mem.U8(srcAddr + 2)
		d.drawSprite(fg, y*DisplayWidth+x, s, false)
		srcAddr += 3
	}

	d.foregroundDirty = true
}

// drawSprite draws sprite n to the specified buffer at the given address.
func (d *Device) drawSprite(dst []byte, dstAddr, n int, isBackground bool) {
	src := d.sprites[n*internalSpriteByteSize:]

	if isBackground {
		// 'fast' lane for background sprites.
		for y := 0; y < SpritePixelSize; y++ {
			dy := dstAddr + y*DisplayWidth
			sy := y * SpritePixelSize
			copy(dst[dy:], src[sy:sy+SpritePixelSize])
		}
	} else {
		// Foreground sprites should refer to color indices in the upper half of the color palette.
		// Each pixel in a foreground sprite must therefore be offset by the appropriate size.
		//
		// Instead of adding the palette offset to each sprite pixel individually, we create an 8-byte
		// wide mask that just repeats the offset 8 times. Once for each pixel in a sprite row. We then
		// OR the mask with each row in one go.
		const PaletteSizeRow = PaletteSize<<56 | PaletteSize<<48 | PaletteSize<<40 | PaletteSize<<32 |
			PaletteSize<<24 | PaletteSize<<16 | PaletteSize<<8 | PaletteSize

		for y := 0; y < SpritePixelSize; y++ {
			dy := dstAddr + y*DisplayWidth
			sy := y * SpritePixelSize
			copy(dst[dy:], src[sy:sy+SpritePixelSize])
			*(*uint64)(unsafe.Pointer(&dst[dy:][0])) |= PaletteSizeRow
		}
	}
}

func (d *Device) clearBackground() {
	copy(d.background[:], d.empty[:])
	d.backgroundDirty = true
}

func (d *Device) clearForeground() {
	copy(d.foreground[:], d.empty[:])
	d.foregroundDirty = true
}

func (d *Device) setBackgroundSprites(mem devices.Memory) {
	address := mem.U16(cpu.R1)
	index := mem.U16(cpu.R2)
	count := mem.U16(cpu.R3)
	d.setSprites(mem, address, index, count)
}

func (d *Device) setForegroundSprites(mem devices.Memory) {
	address := mem.U16(cpu.R1)
	index := BufferSize + mem.U16(cpu.R2)
	count := mem.U16(cpu.R3)
	d.setSprites(mem, address, index, count)
}

func (d *Device) setSprites(src devices.Memory, address, index, count int) {
	dsti := index * internalSpriteByteSize
	dst := d.sprites[:]

	for srci := address; srci < address+count*SpriteByteSize; srci++ {
		bb := src.U8(srci)
		dst[dsti+0] = byte(bb >> 4)
		dst[dsti+1] = byte(bb & 0xf)
		dsti += 2
	}
}

func dumpSprite4bpp(mem devices.Memory, address int) {
	const Stride = SpritePixelSize >> 1
	var row [Stride]byte
	for y := 0; y < SpritePixelSize; y++ {
		mem.Read(address+y*Stride, row[:])
		fmt.Printf("%02x\n", row[:])
	}
	fmt.Println()
}

func dumpSprite8bpp(p []byte) {
	for y := 0; y < SpritePixelSize; y++ {
		fmt.Printf("%02x\n", p[y*SpritePixelSize:y*SpritePixelSize+SpritePixelSize])
	}
	fmt.Println()
}

func (d *Device) setBackgroundPalette(mem devices.Memory) {
	addr := mem.U16(cpu.R1)
	pal := d.palette[:PaletteSize*4]
	pal[3] = 0 // First color is always transparent.

	for i := 1; i < PaletteSize; i++ {
		n2f(mem.U16(addr+i*2), pal[i*4:])
	}

	d.paletteDirty = true
}

func (d *Device) setForegroundPalette(mem devices.Memory) {
	addr := mem.U16(cpu.R1)
	pal := d.palette[PaletteSize*4:]
	pal[3] = 0 // First color is always transparent.

	for i := 1; i < PaletteSize; i++ {
		n2f(mem.U16(addr+i*2), pal[i*4:])
	}

	d.paletteDirty = true
}

// f2n coverts the RGBA color in p to its 16-bit RGB565 equivalent.
func f2n(p []float32) int {
	return int(p[0]*0.31)<<11 | int(p[1]*0.63)<<5 | int(p[2]*0.31)
}

// n2f sets p to the RGBA8 representation RGB565 color in n.
func n2f(n int, p []float32) {
	p[0] = float32((n>>11)&31) / 31
	p[1] = float32((n>>5)&63) / 63
	p[2] = float32(n&31) / 31
	p[3] = 1
}

var quadVertices = []float32{
	//  X, Y, Z, U, V
	-1.0, -1.0, 0.0, 0.0, 1.0,
	1.0, -1.0, 0.0, 1.0, 1.0,
	-1.0, 1.0, 0.0, 0.0, 0.0,
	1.0, -1.0, 0.0, 1.0, 1.0,
	1.0, 1.0, 0.0, 1.0, 0.0,
	-1.0, 1.0, 0.0, 0.0, 0.0,
}
