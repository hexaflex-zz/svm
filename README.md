## SVM

__note__: This project is strictly for personal educational purposes. Don't expect it
to be of any practical use.

SVM implements a programmable virtual machine for a fictional 16-bit architecture.
It comes with an assembler and the runtime. The runtime contains the CPU and a
number of virtual hardware peripherals which one can interact with through a 
program. These include, among other things, a sprite-based display and a gamepad.

Refer to the documentation in the `docs` directory for details on the assembly
language, the architecture and how to interact with peripherals.

### Directory Overview

* __arch__: A small, shared package which defines the CPU architecture. Including the
  instruction set and registers.
* __asm__: Implements the assembler.
  * __asm/ar__: Implements the compiled binary file format. Archives are what the
    assembler produces.
  * __asm/eval__: A helper package for the assembler. It evaluates compile-time expressions.
  * __asm/parser__: The tokenizer and AST builder for the asembler. It reads SVM source files
    and parses them into an Abstract Syntax Tree.
  * __asm/syntax__: A helper package for the assembler. It examines a newly parsed AST and
    ensures it does not contain syntax errors. Additionally performs translations of
    certain code constructs.
* __cmd__: Contains executables. These are mainly front-ends for packages and some useful tools.
  * __cmd/svm__: Contains the executable VM. This is the one thay actually runs your programs.
  * __cmd/svm-asm__: Contains the executable front-end for the assembler.
  * __cmd/svm-fdd__: A small program which creates and examines 1.44MB floppy disk images.
  * __cmd/svm-sprite__: A small tool which generates SVM source code from sprite sheets.
* __devices__: The root directory for implementations of all the virtual hardware components.
  As well as defining some common shared interface types.
  * __devices/fffe/cpu__: Implements the CPU that runs the code.
  * __devices/fffe/gp14__: Implements a virtual gamepad. It exposes a real gamepad to VM code.
  * __devices/fffe/sprdi__: Implements a virtual display. It allows a program to render sprites.
* __docs__: Contains text files with documentation for various components.
* __testdata__: Contains sample SVM source code and some other testing things.


### TODO

* Allow devices to trigger interrupts on the CPU.
  * Implement interrupt queue in CPU.
* Add audio support.
  * Create a specification for a audio hardware.
  * Create a device implementation of the specification.
  * Create test code to illustrate use of the device.
* Add network support?
  * Create a specification for a network adapter.
  * Create a device implementation of the specification.
  * Create test code to illustrate use of the device.


## License

Unless otherwise stated, this project and its contents are provided under a 3-Clause BSD license.
Refer to the LICENSE file for its contents.