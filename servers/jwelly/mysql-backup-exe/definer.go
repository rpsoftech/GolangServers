package main

import (
	"bytes"
	"io"
	"regexp"
)

var definerRegex = regexp.MustCompile("DEFINER=(`(?:[^`]|``)*`|'(?:[^']|'')*'|[^@\\s]+)@(`(?:[^`]|``)*`|'(?:[^']|'')*'|\\S+)\\s*")

// stripDefiner removes the DEFINER clause, so the dump restores on a server
// that does not have the original user. Objects then belong to the user
// running the restore. Only the first clause (the object header) is removed,
// never text inside the body.
func stripDefiner(stmt string) string {
	loc := definerRegex.FindStringIndex(stmt)
	if loc == nil {
		return stmt
	}
	return stmt[:loc[0]] + stmt[loc[1]:]
}

// maxHeaderLine caps how much of a candidate header line definerFilter holds
// in memory. Longer lines pass through unchanged.
const maxHeaderLine = 64 << 10

// definerFilter strips DEFINER clauses from mysqldump output as it streams.
//
// mysqldump writes the clause on the header line of each object, which starts
// with "CREATE" or a "/*!" version comment, for example:
//
//	CREATE DEFINER=`root`@`localhost` PROCEDURE `p`()
//	/*!50013 DEFINER=`root`@`localhost` SQL SECURITY DEFINER */
//	/*!50003 CREATE*/ /*!50017 DEFINER=`root`@`localhost`*/ /*!50003 TRIGGER ...
//
// Only those lines are rewritten. Every other line, including INSERT rows
// that may hold the same text as data, passes through untouched without being
// buffered.
type definerFilter struct {
	w    io.Writer
	line []byte // start of the current line while it may be a header
	pass bool   // the current line is not a header; copy it to the newline
}

func (f *definerFilter) Write(p []byte) (int, error) {
	n := len(p)
	for len(p) > 0 {
		if f.pass {
			i := bytes.IndexByte(p, '\n')
			if i < 0 {
				_, err := f.w.Write(p)
				return n, err
			}
			if _, err := f.w.Write(p[:i+1]); err != nil {
				return n, err
			}
			p = p[i+1:]
			f.pass = false
			continue
		}
		// Take the rest of the line, but never hold more than maxHeaderLine.
		room := maxHeaderLine - len(f.line)
		end, complete := min(len(p), room), false
		if i := bytes.IndexByte(p, '\n'); i >= 0 && i < room {
			end, complete = i+1, true
		}
		f.line = append(f.line, p[:end]...)
		p = p[end:]
		if complete {
			if err := f.flushLine(true); err != nil {
				return n, err
			}
			continue
		}
		if !f.mayBeHeader() || len(f.line) >= maxHeaderLine {
			if err := f.flushLine(false); err != nil {
				return n, err
			}
			f.pass = true
		}
	}
	return n, nil
}

// mayBeHeader reports whether the buffered line can still turn out to be an
// object header.
func (f *definerFilter) mayBeHeader() bool {
	for _, prefix := range [][]byte{[]byte("CREATE"), []byte("/*!")} {
		k := min(len(f.line), len(prefix))
		if bytes.Equal(f.line[:k], prefix[:k]) {
			return true
		}
	}
	return false
}

// flushLine writes the buffered line, stripping DEFINER when the line is
// complete and is a header.
func (f *definerFilter) flushLine(complete bool) error {
	out := f.line
	if complete && f.mayBeHeader() {
		out = []byte(stripDefiner(string(out)))
	}
	f.line = f.line[:0]
	_, err := f.w.Write(out)
	return err
}

// Flush writes a last line that had no trailing newline.
func (f *definerFilter) Flush() error {
	if len(f.line) == 0 {
		return nil
	}
	return f.flushLine(true)
}
