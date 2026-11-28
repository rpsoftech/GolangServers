package cli

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"sync"
	"time"

	"github.com/rpsoftech/golang-servers/servers/pm2go/common"
)

// lastLines returns up to n trailing lines of a file and the offset after them.
func lastLines(path string, n int) ([]string, int64) {
	f, err := os.Open(path)
	if err != nil {
		return nil, 0
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return nil, 0
	}
	size := st.Size()
	const maxRead = 1 << 20
	start := int64(0)
	if size > maxRead {
		start = size - maxRead
	}
	buf := make([]byte, size-start)
	if _, err := f.ReadAt(buf, start); err != nil && err != io.EOF {
		return nil, size
	}
	lines := strings.Split(strings.TrimRight(string(buf), "\n"), "\n")
	if len(lines) == 1 && lines[0] == "" {
		return nil, size
	}
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return lines, size
}

type logSource struct {
	label string
	path  string
	off   int64
}

func showLogs(infos []common.ProcessInfo, lines int, follow, outOnly, errOnly bool) {
	var srcs []*logSource
	for _, i := range infos {
		if !errOnly {
			srcs = append(srcs, &logSource{label: i.Name + " | out", path: i.OutLog})
		}
		if !outOnly {
			srcs = append(srcs, &logSource{label: i.Name + " | err", path: i.ErrLog})
		}
	}
	for _, s := range srcs {
		ls, off := lastLines(s.path, lines)
		s.off = off
		for _, l := range ls {
			fmt.Printf("%s: %s\n", s.label, l)
		}
	}
	if !follow {
		return
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	var mu sync.Mutex
	var wg sync.WaitGroup
	for _, s := range srcs {
		wg.Add(1)
		go func() {
			defer wg.Done()
			followFile(ctx, s, &mu)
		}()
	}
	wg.Wait()
}

func followFile(ctx context.Context, s *logSource, mu *sync.Mutex) {
	t := time.NewTicker(200 * time.Millisecond)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
		f, err := os.Open(s.path)
		if err != nil {
			continue
		}
		st, err := f.Stat()
		if err != nil {
			f.Close()
			continue
		}
		if st.Size() < s.off { // truncated by flush or rotation
			s.off = 0
		}
		if st.Size() > s.off {
			if _, err := f.Seek(s.off, io.SeekStart); err == nil {
				r := bufio.NewReader(f)
				for {
					line, err := r.ReadString('\n')
					if err != nil {
						break // keep a partial last line for the next tick
					}
					s.off += int64(len(line))
					mu.Lock()
					fmt.Printf("%s: %s\n", s.label, strings.TrimRight(line, "\r\n"))
					mu.Unlock()
				}
			}
		}
		f.Close()
	}
}
