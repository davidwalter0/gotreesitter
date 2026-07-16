package gotreesitter

import (
	"fmt"
	"io"
)

// InputReader supplies parser source as a sequence of byte-offset-addressed
// chunks, mirroring tree-sitter's TSInput.read callback
// (payload, byte_offset, position) -> bytes. It lets a caller hand the
// parser a source that is naturally produced incrementally -- an io.Reader,
// a rope/piece-table, a generator, bytes still arriving over a socket --
// without first flattening it into a single []byte by hand.
//
// # Scope
//
// gotreesitter's DFA lexer (Lexer, dfaTokenSource) indexes its source
// []byte directly and at arbitrary offsets -- external scanners in
// particular can look ahead, mark, and reset within already-seen input --
// and a returned Tree permanently retains the full source []byte for
// Node.Text, incremental reparse, and language-specific result
// normalization (see Tree.source in tree.go). Both of those are
// load-bearing across the parser core (parser_dfa_token_source.go and
// external_lexer.go index source[...] directly in dozens of places). So
// unlike upstream tree-sitter's C lexer, which can pull one chunk at a time
// and discard earlier ones, gotreesitter needs the complete source resident
// once parsing starts and for the lifetime of the returned Tree.
//
// Given that, ReadAllInput and the Parse*Reader family below close the API
// gap by fully materializing an InputReader before handing it to the
// existing []byte-based Parse pipeline. This is still valuable: it lets
// callers describe source as a callback/io.Reader instead of writing their
// own accumulation loop, and it accepts sources that were never resident as
// one contiguous slice in the first place. It does not reduce gotreesitter's
// peak memory use versus calling Parse directly, and it is not the
// lex-one-chunk-at-a-time streaming upstream's C implementation supports;
// that would require the deeper core changes noted above.
type InputReader interface {
	// ReadAt returns the chunk of source bytes starting at byteOffset.
	// position is the Point (row/column) corresponding to byteOffset,
	// provided as a hint for implementations that track it incrementally
	// (mirrors TSInput.read's third argument); it is safe to ignore.
	//
	// A zero-length chunk with a nil error signals end of input, matching
	// TSInput.read's empty-slice-means-EOF contract. The returned slice is
	// copied by the caller before ReadAt is invoked again, so
	// implementations may reuse a buffer across calls.
	ReadAt(byteOffset uint32, position Point) ([]byte, error)
}

// InputReaderFunc adapts a function to an InputReader.
type InputReaderFunc func(byteOffset uint32, position Point) ([]byte, error)

// ReadAt calls f.
func (f InputReaderFunc) ReadAt(byteOffset uint32, position Point) ([]byte, error) {
	return f(byteOffset, position)
}

// defaultInputReaderChunkSize is used by the io.Reader / io.ReaderAt
// adapters below when the caller passes chunkSize <= 0.
const defaultInputReaderChunkSize = 4096

// readerInputReader adapts an io.Reader into an InputReader by pulling up to
// chunkSize bytes per call. io.Reader has no way to seek, so ReadAt calls
// must arrive in strictly increasing, contiguous byteOffset order -- which
// is exactly the access pattern ReadAllInput (and therefore Parser.ParseReader)
// uses. An out-of-order call returns an error.
type readerInputReader struct {
	r          io.Reader
	buf        []byte
	nextOffset uint32
	eof        bool
}

// NewReaderInputReader adapts an io.Reader into an InputReader. Reads pull
// up to chunkSize bytes at a time (chunkSize <= 0 uses a default). The
// returned InputReader only supports strictly sequential ReadAt calls
// starting at offset 0, matching io.Reader's own forward-only contract.
func NewReaderInputReader(r io.Reader, chunkSize int) InputReader {
	if chunkSize <= 0 {
		chunkSize = defaultInputReaderChunkSize
	}
	return &readerInputReader{r: r, buf: make([]byte, chunkSize)}
}

func (rr *readerInputReader) ReadAt(byteOffset uint32, _ Point) ([]byte, error) {
	if byteOffset != rr.nextOffset {
		return nil, fmt.Errorf("gotreesitter: io.Reader-backed InputReader requires sequential ReadAt calls: got byte offset %d, want %d", byteOffset, rr.nextOffset)
	}
	if rr.eof {
		return nil, nil
	}
	// io.ReadAtLeast with min=1 retries a Read that legally (if unusually)
	// returns (0, nil) instead of surfacing that as a premature EOF, and
	// normalizes "n>0 then EOF" into (n, nil) so a non-empty final chunk
	// never carries an error here.
	n, err := io.ReadAtLeast(rr.r, rr.buf, 1)
	if n == 0 {
		if err != nil && err != io.EOF {
			return nil, err
		}
		rr.eof = true
		return nil, nil
	}
	rr.nextOffset += uint32(n)
	chunk := make([]byte, n)
	copy(chunk, rr.buf[:n])
	return chunk, nil
}

// readerAtInputReader adapts an io.ReaderAt into an InputReader. Unlike the
// io.Reader adapter, ReadAt calls need not be sequential: each call is
// served directly from the underlying ReaderAt at the requested offset,
// matching TSInput.read's real arbitrary-offset contract.
type readerAtInputReader struct {
	r         io.ReaderAt
	chunkSize int
}

// NewReaderAtInputReader adapts an io.ReaderAt (e.g. *os.File) into an
// InputReader. Reads pull up to chunkSize bytes at a time (chunkSize <= 0
// uses a default). Because io.ReaderAt supports true random access, the
// returned InputReader tolerates ReadAt calls at any byte offset, not only
// sequential ones.
func NewReaderAtInputReader(r io.ReaderAt, chunkSize int) InputReader {
	if chunkSize <= 0 {
		chunkSize = defaultInputReaderChunkSize
	}
	return &readerAtInputReader{r: r, chunkSize: chunkSize}
}

func (ra *readerAtInputReader) ReadAt(byteOffset uint32, _ Point) ([]byte, error) {
	buf := make([]byte, ra.chunkSize)
	n, err := ra.r.ReadAt(buf, int64(byteOffset))
	if n == 0 {
		if err != nil && err != io.EOF {
			return nil, err
		}
		return nil, nil
	}
	if err != nil && err != io.EOF {
		return nil, err
	}
	return buf[:n], nil
}

// ReadAllInput fully materializes r into a single contiguous byte slice by
// calling ReadAt repeatedly with strictly increasing offsets starting at 0
// -- tracking the running Point so each call receives an accurate position
// hint -- until it returns a zero-length chunk (EOF) or an error.
//
// This is the bridge between the callback-based InputReader and
// gotreesitter's existing []byte-based Parse/TokenSource pipeline; see the
// InputReader doc comment for why full materialization, rather than
// incremental chunk-at-a-time lexing, is the current integration point.
func ReadAllInput(r InputReader) ([]byte, error) {
	if r == nil {
		return nil, nil
	}
	var out []byte
	var offset uint32
	var pos Point
	for {
		chunk, err := r.ReadAt(offset, pos)
		if err != nil {
			return nil, err
		}
		if len(chunk) == 0 {
			return out, nil
		}
		out = append(out, chunk...)
		pos = advancePointByBytes(pos, chunk)
		offset += uint32(len(chunk))
	}
}

// ParseInputReader parses source obtained from r, a byte-callback input that
// mirrors tree-sitter's TSInput.read. It fully materializes r via
// ReadAllInput before parsing; see the InputReader doc comment for why
// gotreesitter's DFA lexer and Tree require the complete source rather than
// supporting direct chunk-at-a-time lexing.
//
// Use it when source is naturally produced in chunks (io.Reader, a network
// stream, a rope/piece-table, a generator) and pre-flattening it into a
// []byte yourself would be inconvenient. It does not reduce peak memory use
// versus calling Parse directly with an already-materialized []byte.
func (p *Parser) ParseInputReader(r InputReader) (*Tree, error) {
	source, err := ReadAllInput(r)
	if err != nil {
		return nil, err
	}
	return p.Parse(source)
}

// ParseInputReaderStrict is like ParseInputReader, but returns
// ErrParseStoppedEarly when parsing returns a partial tree.
func (p *Parser) ParseInputReaderStrict(r InputReader) (*Tree, error) {
	return strictParseResult(p.ParseInputReader(r))
}

// ParseReader is a convenience wrapper over ParseInputReader for io.Reader
// sources, reading in chunkSize increments (chunkSize <= 0 uses a default).
func (p *Parser) ParseReader(r io.Reader, chunkSize int) (*Tree, error) {
	return p.ParseInputReader(NewReaderInputReader(r, chunkSize))
}

// ParseReaderStrict is like ParseReader, but returns ErrParseStoppedEarly
// when parsing returns a partial tree.
func (p *Parser) ParseReaderStrict(r io.Reader, chunkSize int) (*Tree, error) {
	return strictParseResult(p.ParseReader(r, chunkSize))
}

// ParseReaderAt is a convenience wrapper over ParseInputReader for
// io.ReaderAt sources (e.g. *os.File), reading in chunkSize increments
// (chunkSize <= 0 uses a default).
func (p *Parser) ParseReaderAt(r io.ReaderAt, chunkSize int) (*Tree, error) {
	return p.ParseInputReader(NewReaderAtInputReader(r, chunkSize))
}

// ParseReaderAtStrict is like ParseReaderAt, but returns
// ErrParseStoppedEarly when parsing returns a partial tree.
func (p *Parser) ParseReaderAtStrict(r io.ReaderAt, chunkSize int) (*Tree, error) {
	return strictParseResult(p.ParseReaderAt(r, chunkSize))
}
