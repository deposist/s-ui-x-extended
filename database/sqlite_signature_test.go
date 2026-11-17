package database

import (
	"bytes"
	"io"
	"testing"
	"testing/iotest"
)

func TestIsSQLiteDBAcceptsShortReads(t *testing.T) {
	for _, tc := range []struct {
		name   string
		reader func(io.Reader) io.Reader
	}{
		{"one byte", iotest.OneByteReader},
		{"half buffer", iotest.HalfReader},
		{"data and EOF", iotest.DataErrReader},
	} {
		t.Run(tc.name, func(t *testing.T) {
			valid, err := IsSQLiteDB(tc.reader(bytes.NewReader([]byte("SQLite format 3\x00"))))
			if err != nil || !valid {
				t.Fatalf("complete SQLite signature rejected after a legal short read: valid=%v, err=%v", valid, err)
			}
		})
	}
}
