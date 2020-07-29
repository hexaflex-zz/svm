## svm-fdd

This tool allows creation and examination of 1.44MB floppy disk image files.
These image files are suitable for use with the FD35 disk drive.


### Example

* Generate an image file `test.fdd` with 1.44MB worth of empty space:

    `$ svm-fdd -out test.fdd`

* Generate an image file `test.fdd` from one or more files:

    `$ svm-fdd -out test.fdd test.a resource1.bin resource2.bin`

  Note that in this case, the resulting image file will be truncated to
  1.44MB if the combined size of the inputs exceeds this size. The files
  are stored in the order they are specified in the command line.


### Supported options

    $ svm-fdd [options] [<input files>]
    -out string
            Output file to generate (Not optional).
    -version
            Display version information.

