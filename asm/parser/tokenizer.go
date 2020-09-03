package parser

import (
	"fmt"
	"io"
	"io/ioutil"
	"runtime"
)

// Known token types.
const (
	tokInstructionBegin = 1 + iota
	tokInstructionEnd
	tokMacroBegin
	tokMacroEnd
	tokExpressionBegin
	tokExpressionEnd
	tokScopeBegin
	tokScopeEnd
	tokIfBegin
	tokIfEnd
	tokLabel
	tokNumber
	tokIdent
	tokString
	tokChar
	tokOperator
	tokAddressMode
	tokBreakPoint
	tokTypeDescriptor
)

// tokenFunc is called whenever a new token is read from source.
type tokenFunc func(typ int, pos Position, value string) error

// tokenizer defines tokenizer state.
type tokenizer struct {
	lineSizes []int
	data      []byte
	tf        tokenFunc
	start     Position
	end       Position
	atEOF     byte
}

// tokenize reads sourcecode from the given reader and turns it into a flat
// stream of tokens. Each token is passed into the given TokenFunc as it
// is read. The filename provides source context for each token.
func tokenize(r io.Reader, filename string, tf tokenFunc) (err error) {
	var tok tokenizer

	tok.data, err = ioutil.ReadAll(r)
	if err != nil {
		return fmt.Errorf("parse error: %v", err)
	}

	//tok.data = append(tok.data, '\n')

	// The tokenizer breaks out of its loop through the use of a panic,
	// We need to catch it here and convert it to a proper error message.
	defer func() {
		x := recover()
		if x == nil || x == io.EOF {
			return
		}

		if _, ok := x.(runtime.Error); ok {
			panic(x)
		}

		err = x.(error)
	}()

	tok.tf = tf
	tok.start = Position{
		File: filename,
		Line: 1,
		Col:  1,
	}
	tok.end = tok.start
	tok.readDocument()
	return
}

// readDocument reads a source file.
func (t *tokenizer) readDocument() {
	for {
		switch {
		case t.readCode():
		default:
			t.error("unexpected token: '%c'; expected comment, label or instruction", t.read())
		}
	}
}

// readDocumentElement reads any valid top-level construct.
func (t *tokenizer) readCode() bool {
	switch {
	case t.readSpace():
	case t.readComment():
	case t.readScope():
	case t.readLabel():
	case t.readBreakpoint():
	case t.readIf():
	case t.readMacro():
	case t.readInstruction():
	default:
		return false
	}
	return true
}

// readBreakpoint reads a breakpoint instruction.
func (t *tokenizer) readBreakpoint() bool {
	if !t.readWord("break") {
		return false
	}
	t.emit(tokBreakPoint)
	return true
}

// readIf reads an if statement.
func (t *tokenizer) readIf() bool {
	if !t.readWord("if") {
		return false
	}

	t.emit(tokIfBegin)
	defer t.emit(tokIfEnd)

	t.readExpression()
	if !t.readInstruction() {
		t.error("if statement must be followed by an instruction")
	}

	return true
}

// readScope reads a scope block. This covers zero or more instructions, labels or other
// constructs, encased in '{' and '}'.
func (t *tokenizer) readScope() bool {
	t.readSpace()
	if !t.readChar('{') {
		return false
	}

	t.emit(tokScopeBegin)
	defer t.emit(tokScopeEnd)

loop:
	for {
		switch {
		case t.readChar('}'):
			break loop
		case t.readCode():
		default:
			t.error("unexpected token in scope block: '%c'; expected comment, label, instruction, scope block or '}'", t.read())
		}
	}

	return true
}

// readLabel reads a label definition.
func (t *tokenizer) readLabel() bool {
	if !t.readChar(':') {
		return false
	}

	t.ignore()

	if !t.readName() {
		t.error("invalid label definition; expected name")
	}

	t.emit(tokLabel)
	return true
}

// readMacro reads a full macro definition.
func (t *tokenizer) readMacro() bool {
	if !t.readWord("macro") {
		return false
	}

	t.ignore()
	t.readSpace()

	if !t.readName() {
		t.error("unexpected token %c; expected macro name", t.read())
	}

	t.emit(tokMacroBegin)
	defer t.emit(tokMacroEnd)

	for t.readExpression() {
	}

loop:
	for {
		switch {
		case t.readWord("endmacro"):
			t.ignore()
			break loop
		case t.readCode():
		default:
			t.error("unexpected token in macro body: '%c'; expected comment, label, instruction, scope block or 'endmacro'", t.read())
		}
	}

	return true
}

// readInstruction reads a full instruction definition.
func (t *tokenizer) readInstruction() bool {
	if !t.readName() {
		t.unread(-1)
		return false
	}

	t.emit(tokInstructionBegin)
	defer t.emit(tokInstructionEnd)

	for t.readExpression() {
	}

	return true
}

// readExpression reads an expression.
// Returns true if a comma (or '=') is encountered, meaning more expressions follow.
func (t *tokenizer) readExpression() bool {
	comma := t.readChar(',') || t.readChar('=')
	t.ignore()

	if t.readSpace() && !comma {
		return false
	}

	t.emit(tokExpressionBegin)
	defer t.emit(tokExpressionEnd)

	for {
		switch {
		case t.readSpace():
			return false
		case t.readComment():
			return false
		case t.readWord("$$"):
			t.emit(tokIdent)
		case t.readWord("$"):
			t.emit(tokAddressMode)
		case t.readChar(','), t.readChar('='):
			t.unread(1)
			return true
		case t.readTypeDescriptor():
		case t.readOperator():
		case t.readValue():
		default:
			t.error("unexpected token '%c'; want comma, operator or value", t.read())
		}
	}
}

// readTypeDescriptor reads a type descriptor at the start of an instruction operand.
func (t *tokenizer) readTypeDescriptor() bool {
	switch {
	case t.readUniqueWord("u8"):
	case t.readUniqueWord("i8"):
	case t.readUniqueWord("u16"):
	case t.readUniqueWord("i16"):
	default:
		return false
	}

	t.emit(tokTypeDescriptor)
	return true
}

// readValue reads a single expression value.
func (t *tokenizer) readValue() bool {
	return t.readNumber() || t.readIdent() || t.readCharlit() || t.readString()
}

// readOperator reads an operator in an expression.
func (t *tokenizer) readOperator() bool {
	r := t.read()
	t.read()

	// Deal with multi-byte operators seperately.
	switch t.current() {
	case ">>", "<<", "!=", "==", "<=", ">=":
		t.emit(tokOperator)
		return true
	}

	t.unread(1)

	if isOperator(r) {
		t.emit(tokOperator)
		return true
	}

	t.unread(1)
	return false
}

// readCharlit reads a character literal. This supports escape sequences.
func (t *tokenizer) readCharlit() bool {
	if !t.readChar('\'') {
		return false
	}

	var escaping bool

loop:
	for {
		r := t.read()

		switch r {
		case '\\':
			escaping = !escaping
		case '\'':
			if !escaping {
				break loop
			}
			escaping = false
		default:
			escaping = false
		}
	}

	t.emit(tokChar)
	return true
}

// readString reads a string literal. This supports escape sequences.
func (t *tokenizer) readString() bool {
	if !t.readChar('"') {
		return false
	}

	var escaping bool

loop:
	for {
		r := t.read()

		switch r {
		case '\\':
			escaping = !escaping
		case '"':
			if !escaping {
				break loop
			}
			escaping = false
		default:
			escaping = false
		}
	}

	t.emit(tokString)
	return true
}

// readNumber reads a numeric literal.
//
// A number can take the form: x#y
// Where x is the base number and y is the actual numeric value.
// For instance:
//
//    2#10011010
//    8#644
//    10#123
//    16#ff
//
// The base prefix is optional. Numbers without a prefix default
// to base 10.
func (t *tokenizer) readNumber() bool {
	if t.parseNumber() {
		t.emit(tokNumber)
		return true
	}
	return false
}

func (t *tokenizer) parseNumber() bool {
	// The underscore in this set can be used to make large numbers easier to read.
	// It should be considered a valid digit.
	digits := []byte(`_0123456789abcdefABCDEF`)

	// Read optional sign.
	signed := t.readAny('-', '+')

	// Check if we have a base prefix or not.
	switch {
	case t.readWord("2#"):
		digits = digits[:3]
	case t.readWord("8#"):
		digits = digits[:9]
	case t.readWord("16#"):
		digits = digits[:17]
	default:
		digits = digits[:11]
	}

	// Number must begin with a digit.
	if !t.readAny(digits[1:]...) {
		if signed {
			t.unread(-1)
		}
		return false
	}

	t.readSet(digits...)

	if !t.haveWordDelim() {
		t.error("unexpected token '%c'; expected whitespace or any of %q", t.read(), digits)
	}

	return true
}

// readIdent reads an identifier.
func (t *tokenizer) readIdent() bool {
	if !t.readName() {
		return false
	}
	t.emit(tokIdent)
	return true
}

// readName reads a name.
func (t *tokenizer) readName() bool {
	if r := t.read(); r != '.' && r != '_' && !isAlpha(r) {
		t.unread(1)
		return false
	}

	for {
		r := t.read()
		if isSpace(rune(r)) || !(r == '_' || r == '.' || isAlpha(r) || isDigit(r)) {
			t.unread(1)
			break
		}
	}

	return true
}

// readUniqueWord does the same as readWord, except it ensures that
// the given word is immediately followed by a word boundary character.
func (t *tokenizer) readUniqueWord(str string) bool {
	if !t.readWord(str) {
		return false
	}

	if !t.haveWordDelim() {
		t.unread(-1)
		return false
	}

	return true
}

// readWord reads runes equal to the given string.
// Returns false if there is no match.
func (t *tokenizer) readWord(str string) bool {
	for _, c := range []byte(str) {
		if !t.readChar(c) {
			t.unread(-1)
			return false
		}
	}
	return true
}

// readComment reads and skips code comments.
func (t *tokenizer) readComment() bool {
	if t.read() != ';' {
		t.unread(1)
		return false
	}

	t.readUntil('\n')
	t.ignore()
	return true
}

// readSpace reads whitespace and skips it.
// Returns true if a newline character was encountered.
func (t *tokenizer) readSpace() bool {
	var r byte
	var newline bool

	for r = t.read(); isSpace(rune(r)); r = t.read() {
		if r == '\n' {
			newline = true
		}
	}

	if t.atEOF == 0 {
		t.unread(1)
	}

	t.ignore()
	return newline
}

// haveWordDelim checks if the next byte constitutes a word delimiter.
// This would typically be whitespace or a comma.
func (t *tokenizer) haveWordDelim() bool {
	r := t.read()
	defer t.unread(1)

	switch {
	case r == ',':
	case isOperator(r):
	case isSpace(rune(r)):
	default:
		return false
	}

	return true
}

// readSet reads bytes as long as they occur in set.
// Returns true if more than zero bytes have been read.
func (t *tokenizer) readSet(set ...byte) bool {
	var n int

	for inSet(set, t.read()) {
		n++
	}

	t.unread(1)
	return n > 0
}

// readUntil reads bytes until it encounters x.
// Returns true if more than zero bytes have been read.
func (t *tokenizer) readUntil(x byte) bool {
	var n int

	for t.read() != x {
		n++
	}

	t.unread(1)
	return n > 0
}

// readAny reads the next byte if it is in the given set.
func (t *tokenizer) readAny(set ...byte) bool {
	if len(set) == 0 {
		return false
	}

	if inSet(set, t.read()) {
		return true
	}

	t.unread(1)
	return false
}

// readChar reads the next byte, only if it matches x.
func (t *tokenizer) readChar(x byte) bool {
	if t.read() == x {
		return true
	}
	t.unread(1)
	return false
}

// current returns the current read token.
func (t *tokenizer) current() string {
	return string(t.data[t.start.Offset:t.end.Offset])
}

// error emits a new error token with the given message.
func (t *tokenizer) error(f string, argv ...interface{}) {
	panic(NewError(t.start, fmt.Sprintf(f, argv...)))
}

// emit emits a new token of the given type, using the currently
// read buffer.
func (t *tokenizer) emit(typ int) {
	value := t.current()

	if err := t.tf(typ, t.start, value); err != nil {
		panic(err)
	}

	t.ignore()
}

// ignore skips the currently read buffer.
func (t *tokenizer) ignore() {
	t.start = t.end
}

// unread unreads the last n read bytes.
// This can not read back into the previous token.
// If n is -1, this unreads the entire token.
func (t *tokenizer) unread(n int) {
	if n == -1 {
		t.end = t.start
		return
	}

	var r byte
	for ; n > 0; n-- {
		t.end.Offset--
		if t.end.Offset >= len(t.data) {
			r = '\n'
		} else {
			r = t.data[t.end.Offset]
		}

		if r == '\n' {
			t.end.Line--
			t.end.Col = t.lineSizes[len(t.lineSizes)-1]
			t.lineSizes = t.lineSizes[:len(t.lineSizes)-1]
		} else {
			t.end.Col--
		}
	}
}

// read reads the next byte from the stream.
func (t *tokenizer) read() byte {
	var r byte

	if t.end.Offset >= len(t.data) {
		if t.atEOF > 3 {
			panic(io.EOF)
		}

		t.atEOF++
		r = '\n'
	} else {
		r = t.data[t.end.Offset]
	}

	t.end.Offset++

	if r == '\n' {
		t.lineSizes = append(t.lineSizes, t.end.Col)
		t.end.Line++
		t.end.Col = 1
	} else {
		t.end.Col++
	}

	return r
}

func isAlpha(x byte) bool {
	return (x >= 'a' && x <= 'z') || (x >= 'A' && x <= 'Z')
}

func isDigit(x byte) bool {
	return x >= '0' && x <= '9'
}

func inSet(set []byte, x byte) bool {
	for _, v := range set {
		if x == v {
			return true
		}
	}
	return false
}

func isOperator(x byte) bool {
	switch x {
	case '+', '-', '*', '/', '%', '&', '|', '^', '<', '>', '(', ')':
		return true
	}
	return false
}
