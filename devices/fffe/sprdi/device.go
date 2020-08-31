// Package sprdi implements the Sprite Display Mk I
package sprdi

import (
	"fmt"
	"log"

	"github.com/go-gl/gl/v4.2-core/gl"
	"github.com/pkg/errors"

	"github.com/hexaflex/svm/devices"
	"github.com/hexaflex/svm/devices/fffe/cpu"
)

// Known interrupt operations.
const (
	setPalette = iota
	setSprites
	draw
	clear
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

// Device defines all internal doodads for the display.
type Device struct {
	palette      [PaletteSize * 4]float32
	sprites      [BufferSize * SpritePixelSize * SpritePixelSize]byte
	scene        [DisplayWidth * DisplayHeight]byte
	empty        [DisplayWidth * DisplayHeight]byte
	shader       uint32
	vao          uint32
	vbo          uint32
	sceneTex     uint32
	paletteDirty bool
	sceneDirty   bool
	initialized  bool
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
	gl.BindTexture(gl.TEXTURE_2D, d.sceneTex)

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

	d.sceneTex = makeTexture()

	d.paletteDirty = true
	d.sceneDirty = true
	d.initialized = true
	d.swap()
	return nil
}

// Shutdown clears up device resources.
func (d *Device) Shutdown() error {
	d.initialized = false
	gl.DeleteTextures(1, &d.sceneTex)
	gl.DeleteBuffers(1, &d.vbo)
	gl.DeleteVertexArrays(1, &d.vao)
	gl.DeleteProgram(d.shader)
	return nil
}

// Int triggers an interrupt on the device. The device can read from- and write to system memory.
func (d *Device) Int(mem devices.Memory) {
	switch mem.U16(cpu.R0) {
	case setPalette:
		d.setPalette(mem)
	case setSprites:
		d.setSprites(mem)
	case draw:
		d.draw(mem)
	case clear:
		d.clear()
	case swap:
		d.swap()
	}
}

func (d *Device) swap() {
	if d.paletteDirty {
		gl.UseProgram(d.shader)
		palette := gl.GetUniformLocation(d.shader, glStr("palette"))
		gl.Uniform4fv(palette, 16, &d.palette[0])
		d.paletteDirty = false
	}

	if d.sceneDirty {
		uploadTexture(d.sceneTex, gl.RED, DisplayWidth, DisplayHeight, gl.RED, gl.UNSIGNED_BYTE, d.scene[:])
		d.sceneDirty = false
	}
}

func (d *Device) draw(mem devices.Memory) {
	srcAddr := mem.U16(cpu.R1)
	count := mem.U16(cpu.R2)

	for i := 0; i < count; i++ {
		n := mem.U8(srcAddr + 0)
		x := mem.U8(srcAddr + 1)
		y := mem.U8(srcAddr + 2)
		d.drawSprite(x, y, n)
		srcAddr += 3
	}

	d.sceneDirty = true
}

// drawSprite draws sprite n to the scene buffer at the given address.
func (d *Device) drawSprite(x, y, n int) {
	src := d.sprites[n*internalSpriteByteSize:]
	dst := d.scene[:]
	dstAddr := y*DisplayWidth + x

	for y := 0; y < SpritePixelSize; y++ {
		dy := dstAddr + y*DisplayWidth
		sy := y * SpritePixelSize
		copy(dst[dy:], src[sy:sy+SpritePixelSize])
	}
}

func (d *Device) clear() {
	copy(d.scene[:], d.empty[:])
	d.sceneDirty = true
}

func (d *Device) setSprites(mem devices.Memory) {
	address := mem.U16(cpu.R1)
	index := mem.U16(cpu.R2)
	count := mem.U16(cpu.R3)

	dsti := index * internalSpriteByteSize
	dst := d.sprites[:]

	for srci := address; srci < address+count*SpriteByteSize; srci++ {
		bb := mem.U8(srci)
		dst[dsti+0] = byte(bb >> 4)
		dst[dsti+1] = byte(bb & 0xf)
		dsti += 2
	}

	for i := 0; i < count; i++ {
		x := (index + i) * internalSpriteByteSize
		dumpSprite64(dst[x : x+internalSpriteByteSize])
	}
}

func dumpSprite64(s []byte) {
	for len(s) > 0 {
		fmt.Printf("%02x\n", s[:8])
		s = s[8:]
	}
	fmt.Println()
}

func (d *Device) setPalette(mem devices.Memory) {
	pal := d.palette[:PaletteSize*4]
	addr := mem.U16(cpu.R1)
	pal[3] = 0 // First entry is always transparent.

	for i := 1; i < PaletteSize; i++ {
		r := mem.U8(addr + i*3 + 0)
		g := mem.U8(addr + i*3 + 1)
		b := mem.U8(addr + i*3 + 2)
		n2f(r, g, b, pal[i*4:])
	}

	d.paletteDirty = true
}

// n2f sets converts the given RGB components in the range [0,255] to
// their floating point equivalents and stores them in p.
func n2f(r, g, b int, p []float32) {
	p[0] = float32(r&255) / 255
	p[1] = float32(g&255) / 255
	p[2] = float32(b&255) / 255
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
