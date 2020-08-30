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

uniform vec4 palette[16];

layout (binding = 0) uniform sampler2D scene;

in  vec2 fragTexCoord;
out vec4 outputColor;
 
void main() {   
    uint index = uint(texture2D(scene, fragTexCoord).r * 255) % 16;
    outputColor = palette[index];
}
`
