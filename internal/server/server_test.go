package server

import (
	"fmt"
	"net"
	"net/http"
	"strings"
	"testing"
)

func call(t *testing.T, s *Server, method, path, origin, token, body string) int {
	t.Helper()
	req, _ := http.NewRequest(method, fmt.Sprintf("http://127.0.0.1:%d%s", s.Port(), path), strings.NewReader(body))
	if origin != "" {
		req.Header.Set("Origin", origin)
	}
	req.Header.Set("X-GoDownload-Client", "extension/1.0")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	return resp.StatusCode
}

func TestAPIOriginsAndToken(t *testing.T) {
	var got []Link
	s, err := Start(Config{Ports: []int{0}, OnAdd: func(l []Link, _ bool) { got = append(got, l...) }}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	ext := "chrome-extension://abcdef"
	if c := call(t, s, "GET", "/api/v1/ping", ext, "", ""); c != 200 {
		t.Fatalf("ping = %d", c)
	}
	if c := call(t, s, "POST", "/api/v1/downloads", "https://evil.example", "", `{"url":"https://x/a.zip"}`); c != 403 {
		t.Fatalf("web origin allowed: %d", c)
	}
	if c := call(t, s, "POST", "/api/v1/downloads", ext, "", `{"url":"file:///etc/passwd"}`); c != 400 {
		t.Fatalf("file URL accepted: %d", c)
	}
	if c := call(t, s, "POST", "/api/v1/downloads", ext, "", `{"url":"https://x/a.zip","cookies":"a=1"}`); c != 202 {
		t.Fatalf("add = %d", c)
	}
	if len(got) != 1 || got[0].Cookies != "a=1" {
		t.Fatalf("OnAdd got %+v", got)
	}

	s.SetToken("secret")
	if c := call(t, s, "GET", "/api/v1/ping", ext, "", ""); c != 401 {
		t.Fatalf("missing token = %d", c)
	}
	if c := call(t, s, "GET", "/api/v1/ping", ext, "secret", ""); c != 200 {
		t.Fatalf("with token = %d", c)
	}
}

func TestRanges(t *testing.T) {
	got := ranges([]float32{1, 1, 0.5, 0, 1}, 1000)
	want := [][2]int64{{0, 400}, {800, 1000}}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("ranges = %v", got)
	}
}

func TestFallsBackWhenPortTaken(t *testing.T) {
	blocker, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer blocker.Close()
	taken := blocker.Addr().(*net.TCPAddr).Port
	s, err := Start(Config{Ports: []int{taken, 0}, OnAdd: func([]Link, bool) {}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if s.Port() == taken {
		t.Fatalf("listening on the taken port %d", taken)
	}
}
