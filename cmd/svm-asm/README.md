## svm-asm

This tool reads SVM sourcecode from a given module path and either compiles
it into a binary archive file, or displays a human-readable dump of either
the source AST or the compiled archive.

Refer to `docs/asm.txt` for details on the assembly language and the assembler.
   

## Supported options

    $ svm-asm.exe [options] <target import path>
    -debug
            Include debug symbols in the build.
    -dump-ar
            Print a human-readable version of the compiled binary.
    -dump-ast
            Print a human-readable version of the unprocessed AST.
    -import string
            Root directory for all source code.
    -out string
            Location to store data in. Leave empty to write data to stdout.
    -version
            Display version information.


## Example invocation

    $ svm-asm.exe -import "root" -out "myprogram.bin" myprogram


## Import paths

The `"root"` directory in the example above is where the assembler begins
looking for whichever module is to be compiled. It should house directories 
with the sourcecode for all modules involved in the program.

    [root]
     |- [myprogram]
     |   |-file1.svm
     |   |-file2.svm
     |   |-file3.svm
     |   |-file4.svm
     |- [dependency1]
     |   |-file1.svm
     |   |-file2.svm
     |   |-file3.svm
     |   |-file4.svm
     |- [dependency2]
     |   |-[sub]
     |   |  |-[dir]
     |   |  |  |-file1.svm
     |   |  |  |-file2.svm
     |   |  |  |-file3.svm
     |   |-file1.svm
     |   |-file2.svm

The assembler will automatically collate the source files in a given directory
and consider them part of the same module. Subdirectories are separate modules.
In the listing above, there are 4 unique modules:

   * myprogram
   * dependency1
   * dependency2
   * dependency2/sub/dir
