## svm-sprite

This tool takes an image as input and generates SVM sourcecode as output.
It splits the image up into sprite-sized chunks and generates data statements
for each of them. Effectively embedding all sprites in the input image in
a program which imports the sourcecode.


### Example

    $ svm-sprite -out font.svm font.png

Will generate a file `font.svm` with contents like:

    const SpriteCount = 94

    :Sprites
    d32 16#000ff000
    d32 16#000ff000
    d32 16#000ff000
    d32 16#000ff000
    d32 16#00000000
    d32 16#000ff000
    d32 16#000ff000
    d32 16#00000000

    d32 16#0ff00ff0
    d32 16#0ff00ff0
    d32 16#0ff00ff0
    d32 16#00000000
    d32 16#00000000
    d32 16#00000000
    d32 16#00000000
    d32 16#00000000

    d32 16#0ff0ff00
    d32 16#fffffff0
    d32 16#ff000ff0
    d32 16#0f000f00
    d32 16#ff000ff0
    d32 16#fffffff0
    d32 16#0ff0ff00
    d32 16#00000000

    <snip>