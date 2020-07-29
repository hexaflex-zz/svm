## svm

SVM is the Virtual machine which runs programs compiled with `svm-asm`
and packed into a floppy image with `svm-fdd`.

## Supported options

        $ svm [options] <image file>
        -debug
                Run in debug mode.
        -readonly
                Is the loaded floppy disk write protected?
        -fullscreen
                Run the display in fullscreen or windowed mode.
        -scale-factor int
                Pixel scale factor for the display. (default 2)
        -version
                Display version information.


## Example invocation

    $ svm -debug myprogram.img

