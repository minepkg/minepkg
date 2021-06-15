package resolver

// WriteCounter counts the number of bytes written to it.
type WriteCounter struct {
	Total *uint64 // Total # of bytes transferred
}

// Write implements the io.Writer interface.
func (wc *WriteCounter) Write(p []byte) (int, error) {
	n := len(p)
	*wc.Total += uint64(n)
	return n, nil
}
