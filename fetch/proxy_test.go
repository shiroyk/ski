package fetch

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestSOCKS5Proxy(t *testing.T) {
	defer afterTest(t)
	ch := make(chan string, 1)
	l := newLocalListener(t)
	defer l.Close()
	defer close(ch)
	proxy := func(t *testing.T) {
		s, err := l.Accept()
		if err != nil {
			t.Errorf("socks5 proxy Accept(): %v", err)
			return
		}
		defer s.Close()
		var buf [22]byte
		if _, err := io.ReadFull(s, buf[:3]); err != nil {
			t.Errorf("socks5 proxy initial read: %v", err)
			return
		}
		if want := []byte{5, 1, 0}; !bytes.Equal(buf[:3], want) {
			t.Errorf("socks5 proxy initial read: got %v, want %v", buf[:3], want)
			return
		}
		if _, err := s.Write([]byte{5, 0}); err != nil {
			t.Errorf("socks5 proxy initial write: %v", err)
			return
		}
		if _, err := io.ReadFull(s, buf[:4]); err != nil {
			t.Errorf("socks5 proxy second read: %v", err)
			return
		}
		if want := []byte{5, 1, 0}; !bytes.Equal(buf[:3], want) {
			t.Errorf("socks5 proxy second read: got %v, want %v", buf[:3], want)
			return
		}
		var ipLen int
		switch buf[3] {
		case 1:
			ipLen = net.IPv4len
		case 4:
			ipLen = net.IPv6len
		default:
			t.Errorf("socks5 proxy second read: unexpected address type %v", buf[4])
			return
		}
		if _, err := io.ReadFull(s, buf[4:ipLen+6]); err != nil {
			t.Errorf("socks5 proxy address read: %v", err)
			return
		}
		ip := net.IP(buf[4 : ipLen+4])
		port := binary.BigEndian.Uint16(buf[ipLen+4 : ipLen+6])
		copy(buf[:3], []byte{5, 0, 0})
		if _, err := s.Write(buf[:ipLen+6]); err != nil {
			t.Errorf("socks5 proxy connect write: %v", err)
			return
		}
		ch <- fmt.Sprintf("proxy for %s:%d", ip, port)

		// Implement proxying.
		targetHost := net.JoinHostPort(ip.String(), strconv.Itoa(int(port)))
		targetConn, err := net.Dial("tcp", targetHost)
		if err != nil {
			t.Errorf("net.Dial failed")
			return
		}
		go io.Copy(targetConn, s)
		io.Copy(s, targetConn) // Wait for the client to close the socket.
		targetConn.Close()
	}

	sentinelHeader := "X-Sentinel"
	sentinelValue := "12345"
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(sentinelHeader, sentinelValue)
	})
	for _, useTLS := range []bool{false, true} {
		t.Run(fmt.Sprintf("useTLS=%v", useTLS), func(t *testing.T) {
			var ts *httptest.Server
			if useTLS {
				ts = httptest.NewTLSServer(h)
			} else {
				ts = httptest.NewServer(h)
			}
			go proxy(t)
			c := ts.Client()
			c.Transport.(*http.Transport).Proxy = RoundRobinProxy
			AddRoundRobinProxy(ts.URL, "socks5://"+l.Addr().String())
			r, err := c.Head(ts.URL)
			if err != nil {
				t.Fatal(err)
			}
			if r.Header.Get(sentinelHeader) != sentinelValue {
				t.Errorf("Failed to retrieve sentinel value")
			}
			var got string
			select {
			case got = <-ch:
			case <-time.After(5 * time.Second):
				t.Fatal("timeout connecting to socks5 proxy")
			}
			ts.Close()
			tsu, err := url.Parse(ts.URL)
			if err != nil {
				t.Fatal(err)
			}
			want := "proxy for " + tsu.Host
			if got != want {
				t.Errorf("got %q, want %q", got, want)
			}
		})
	}
}

func TestTransportProxy(t *testing.T) {
	defer afterTest(t)
	testCases := []struct{ httpsSite, httpsProxy bool }{
		{false, false},
		{false, true},
		{true, false},
		{true, true},
	}
	for _, testCase := range testCases {
		httpsSite := testCase.httpsSite
		httpsProxy := testCase.httpsProxy
		t.Run(fmt.Sprintf("httpsSite=%v, httpsProxy=%v", httpsSite, httpsProxy), func(t *testing.T) {
			siteCh := make(chan *http.Request, 1)
			h1 := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				siteCh <- r
			})
			proxyCh := make(chan *http.Request, 1)
			h2 := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				proxyCh <- r
				// Implement an entire CONNECT proxy
				if r.Method == http.MethodConnect {
					hijacker, ok := w.(http.Hijacker)
					if !ok {
						t.Errorf("hijack not allowed")
						return
					}
					clientConn, _, err := hijacker.Hijack()
					if err != nil {
						t.Errorf("hijacking failed")
						return
					}
					res := &http.Response{
						StatusCode: http.StatusOK,
						Proto:      "HTTP/1.1",
						ProtoMajor: 1,
						ProtoMinor: 1,
						Header:     make(http.Header),
					}

					targetConn, err := net.Dial("tcp", r.URL.Host)
					if err != nil {
						t.Errorf("net.Dial(%q) failed: %v", r.URL.Host, err)
						return
					}

					if err := res.Write(clientConn); err != nil {
						t.Errorf("Writing 200 OK failed: %v", err)
						return
					}

					go io.Copy(targetConn, clientConn)
					go func() {
						io.Copy(clientConn, targetConn)
						targetConn.Close()
					}()
				}
			})
			var ts *httptest.Server
			if httpsSite {
				ts = httptest.NewTLSServer(h1)
			} else {
				ts = httptest.NewServer(h1)
			}
			var proxy1, proxy2 *httptest.Server
			if httpsProxy {
				proxy1 = httptest.NewTLSServer(h2)
				proxy2 = httptest.NewTLSServer(h2)
			} else {
				proxy1 = httptest.NewServer(h2)
				proxy2 = httptest.NewServer(h2)
			}

			// If neither server is HTTPS or both are, then c may be derived from either.
			// If only one server is HTTPS, c must be derived from that server in order
			// to ensure that it is configured to use the fake root CA from testcert.go.
			c := proxy1.Client()
			if httpsSite {
				c = ts.Client()
			}
			c.Transport.(*http.Transport).Proxy = RoundRobinProxy
			AddRoundRobinProxy(ts.URL, proxy1.URL, proxy2.URL)

			if _, err := c.Head(ts.URL); err != nil {
				t.Error(err)
			}
			var got *http.Request
			select {
			case got = <-proxyCh:
			case <-time.After(5 * time.Second):
				t.Fatal("timeout connecting to http proxy")
			}
			c.Transport.(*http.Transport).CloseIdleConnections()
			ts.Close()
			proxy1.Close()
			proxy2.Close()
			if httpsSite {
				// First message should be a CONNECT, asking for a socket to the real server,
				if got.Method != http.MethodConnect {
					t.Errorf("Wrong method for secure proxying: %q", got.Method)
				}
				gotHost := got.URL.Host
				pu, err := url.Parse(ts.URL)
				if err != nil {
					t.Fatal("Invalid site URL")
				}
				if wantHost := pu.Host; gotHost != wantHost {
					t.Errorf("Got CONNECT host %q, want %q", gotHost, wantHost)
				}

				// The next message on the channel should be from the site's server.
				next := <-siteCh
				if next.Method != http.MethodHead {
					t.Errorf("Wrong method at destination: %s", next.Method)
				}
				if nextURL := next.URL.String(); nextURL != "/" {
					t.Errorf("Wrong URL at destination: %s", nextURL)
				}
			} else {
				if got.Method != http.MethodHead {
					t.Errorf("Wrong method for destination: %q", got.Method)
				}
				gotURL := got.URL.String()
				wantURL := ts.URL + "/"
				if gotURL != wantURL {
					t.Errorf("Got URL %q, want %q", gotURL, wantURL)
				}
			}
		})
	}
}

func newLocalListener(t *testing.T) net.Listener {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		ln, err = net.Listen("tcp6", "[::1]:0")
	}
	if err != nil {
		t.Fatal(err)
	}
	return ln
}

func interestingGoroutines() (gs []string) {
	buf := make([]byte, 2<<20)
	buf = buf[:runtime.Stack(buf, true)]
	for _, g := range strings.Split(string(buf), "\n\n") {
		_, stack, _ := strings.Cut(g, "\n")
		stack = strings.TrimSpace(stack)
		if stack == "" ||
			strings.Contains(stack, "testing.(*M).before.func1") ||
			strings.Contains(stack, "os/signal.signal_recv") ||
			strings.Contains(stack, "created by net.startServer") ||
			strings.Contains(stack, "created by testing.RunTests") ||
			strings.Contains(stack, "closeWriteAndWait") ||
			strings.Contains(stack, "testing.Main(") ||
			// These only show up with GOTRACEBACK=2; Issue 5005 (comment 28)
			strings.Contains(stack, "vm.goexit") ||
			strings.Contains(stack, "created by vm.gc") ||
			strings.Contains(stack, "interestingGoroutines") ||
			strings.Contains(stack, "vm.MHeap_Scavenger") {
			continue
		}
		gs = append(gs, stack)
	}
	sort.Strings(gs)
	return
}

func afterTest(t testing.TB) {
	http.DefaultTransport.(*http.Transport).CloseIdleConnections()
	if testing.Short() {
		return
	}
	var bad string
	badSubstring := map[string]string{
		").readLoop(":  "a Transport",
		").writeLoop(": "a Transport",
		"created by net/http/httptest.(*Server).Start": "an httptest.Server",
		"timeoutHandler":        "a TimeoutHandler",
		"net.(*netFD).connect(": "a timing out dial",
		").noteClientGone(":     "a closenotifier sender",
	}
	var stacks string
	for i := 0; i < 10; i++ {
		bad = ""
		stacks = strings.Join(interestingGoroutines(), "\n\n")
		for substr, what := range badSubstring {
			if strings.Contains(stacks, substr) {
				bad = what
			}
		}
		if bad == "" {
			return
		}
		// Bad stuff found, but goroutines might just still be
		// shutting down, so give it some time.
		time.Sleep(250 * time.Millisecond)
	}
	t.Errorf("Test appears to have leaked %s:\n%s", bad, stacks)
}
