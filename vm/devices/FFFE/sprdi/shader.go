package sprdi

const vertex = `
#version 420

in  vec3 vertPos;
in  vec2 vertTexCoord;
out vec2 fragTexCoord;

void main() {
    fragTexCoord = vertTexCoord;
    gl_Position  = vec4(vertPos, 1);
}

`
const fragment = `
#version 420

uniform vec4 palette[32];

layout (binding = 0) uniform sampler2D background;
layout (binding = 1) uniform sampler2D foreground;

in  vec2 fragTexCoord;
out vec4 outputColor;
 
void main() {   
    // Read color palette indices from the foreground- and background buffers.
    // They are stored in the respective red channels. We need them to be in
    // the original byte format and in the [0,32) range.
    uint bgp = uint(texture2D(background, fragTexCoord).r * 255) % 32;
    uint fgp = uint(texture2D(foreground, fragTexCoord).r * 255) % 32;

    // Sample background- and foreground colors from palette.
    vec4 bgc = palette[bgp];
    vec4 fgc = palette[fgp];

    // Use fgc's alpha channel to determine which color to assign to the output.
    outputColor = mix(bgc, fgc, fgc.a);
}
`
