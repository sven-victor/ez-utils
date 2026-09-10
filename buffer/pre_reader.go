package buffer

import "io"

// PreReader is a ReadCloser that has already consumed and buffered a prefix
// of the underlying stream. Buffer and String return that prefix.
type PreReader interface {
	io.ReadCloser
	// Buffer returns the prefetched prefix of the stream.
	Buffer() []byte
	// String returns the prefetched prefix as a string.
	String() string
}

type preReader struct {
	io.ReadCloser
	buf    []byte
	bufIdx int
	maxLen int
}

func (r *preReader) Close() error {
	r.buf = nil
	r.bufIdx = 0
	r.maxLen = 0
	return r.ReadCloser.Close()
}

func (r *preReader) Buffer() []byte {
	return r.buf
}

func (r *preReader) String() string {
	return string(r.buf)
}

func (r *preReader) Read(p []byte) (n int, err error) {
	if r.bufIdx >= r.maxLen {
		return r.ReadCloser.Read(p)
	} else if len(r.buf) <= r.bufIdx {
		return 0, io.EOF
	}
	n = copy(p, r.buf[r.bufIdx:])
	r.bufIdx += n
	return
}

// NewPreReader reads up to maxLen bytes from reader, buffers them, and
// returns a PreReader that replays that prefix followed by the rest of reader.
func NewPreReader(reader io.ReadCloser, maxLen int) (PreReader, error) {
	r := &preReader{ReadCloser: reader, maxLen: maxLen}
	for i := 0; i < maxLen; {
		p2 := make([]byte, maxLen-i)
		if n, err := reader.Read(p2); n != 0 {
			r.buf = append(r.buf, p2[:n]...)
			i += n
		} else if err == io.EOF {
			break
		} else if err != nil {
			return r, err
		}
	}
	return r, nil
}
