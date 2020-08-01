package sprdi

import (
	"fmt"
	"strings"

	"github.com/go-gl/gl/v4.2-core/gl"
	"github.com/pkg/errors"
)

// makeTexture creates a new texture.
func makeTexture() uint32 {
	var tex uint32
	gl.GenTextures(1, &tex)
	gl.ActiveTexture(gl.TEXTURE_2D)
	gl.BindTexture(gl.TEXTURE_2D, tex)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.NEAREST)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.NEAREST)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE)
	return tex
}

// glStr returns v as a C string, suitable for use with opengl.
func glStr(v string) *uint8 {
	return gl.Str(v + "\x00")
}

func uploadTexture(texture uint32, internalformat, width, height int32, format, xtype uint32, pixels []byte) {
	gl.ActiveTexture(gl.TEXTURE0)
	gl.BindTexture(gl.TEXTURE_2D, texture)
	gl.TexImage2D(gl.TEXTURE_2D, 0, internalformat, width, height, 0, format, xtype, gl.Ptr(pixels))
}

// compileProgram compiles the given shader sources into a program.
func compileProgram(vertex, geometry, fragment string) (uint32, error) {
	var vs, gs, fs uint32
	var err error

	if len(vertex) > 0 {
		vs, err = compileShader(vertex, gl.VERTEX_SHADER)
		if err != nil {
			return 0, errors.Wrapf(err, "failed to compile vertex shader")
		}

		defer gl.DeleteShader(vs)
	}

	if len(geometry) > 0 {
		gs, err = compileShader(geometry, gl.GEOMETRY_SHADER)
		if err != nil {
			return 0, errors.Wrapf(err, "failed to compile geometry shader")
		}

		defer gl.DeleteShader(gs)
	}

	if len(fragment) > 0 {
		fs, err = compileShader(fragment, gl.FRAGMENT_SHADER)
		if err != nil {
			return 0, errors.Wrapf(err, "failed to compile fragment shader")
		}

		defer gl.DeleteShader(fs)
	}

	program := gl.CreateProgram()

	if len(vertex) > 0 {
		gl.AttachShader(program, vs)
	}

	if len(geometry) > 0 {
		gl.AttachShader(program, gs)
	}

	if len(fragment) > 0 {
		gl.AttachShader(program, fs)
	}

	gl.LinkProgram(program)

	var status int32
	gl.GetProgramiv(program, gl.LINK_STATUS, &status)
	if status == gl.FALSE {
		var logLength int32
		gl.GetProgramiv(program, gl.INFO_LOG_LENGTH, &logLength)

		log := strings.Repeat("\x00", int(logLength+1))
		gl.GetProgramInfoLog(program, logLength, nil, gl.Str(log))

		return 0, fmt.Errorf("failed to link program: %v", log)
	}

	return program, nil
}

// compileShader compiles the given shader source into a program.
func compileShader(source string, stype uint32) (uint32, error) {
	shader := gl.CreateShader(stype)

	csources, free := gl.Strs(source + "\x00")
	gl.ShaderSource(shader, 1, csources, nil)
	free()
	gl.CompileShader(shader)

	var status int32
	gl.GetShaderiv(shader, gl.COMPILE_STATUS, &status)
	if status == gl.FALSE {
		var logLength int32
		gl.GetShaderiv(shader, gl.INFO_LOG_LENGTH, &logLength)

		log := strings.Repeat("\x00", int(logLength+1))
		gl.GetShaderInfoLog(shader, logLength, nil, gl.Str(log))

		return 0, fmt.Errorf("failed to compile %v: %v", source, log)
	}

	return shader, nil
}
