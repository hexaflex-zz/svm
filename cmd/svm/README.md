## svm

SVM is the Virtual machine which runs programs compiled with `svm-asm`.

## Supported options

        $ svm.exe [options] <program>
        -debug
                Run in debug mode.
        -fdd-img string
                File containing floppy disk image.
        -fdd-wp
                Is the loaded floppy disk write protected?
        -fullscreen
                Run the display in fullscreen or windowed mode.
        -scale-factor int
                Pixel scale factor for the display. (default 2)
        -version
                Display version information.


## Example invocation

    $ svm.exe -debug "myprogram.bin"

