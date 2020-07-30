## svm-asm

This tool reads SVM sourcecode from a given module path and either compiles
it into a binary archive, or displays a human-readable dump of either the
source AST or the compiled archive.

Refer to `docs/asm.txt` for details on the assembly language and the assembler.
   

## Supported options

        $ svm-asm [options] <target import path>
        -debug
                Include debug symbols in the build. Creates an extra <out>.dbg file as output.
        -dump-ar
                Print a human-readable version of the compiled binary to stdout.
        -dump-ast
                Print a human-readable version of the unprocessed AST to stdout.
        -import string
                Root directory for all source code.
        -out string
                Output file.
        -version
                Display version information.


## Example invocation

Generate a compiled program in the file `myprogram.bin`, by compiling the
sources from the `myprogram` module. Creates an additional output file
`myprogram.dbg` with debug symbols.

        $ svm-asm -import "root" -out myprogram.bin -debug myprogram


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
