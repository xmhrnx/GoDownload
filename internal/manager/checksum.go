package manager

import (
	"context"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"os"
	"strings"
)

// ParseChecksum normalizes "sha256:ABC…" or a bare hex digest (the algorithm
// guessed from its length) to "algo:hex".
func ParseChecksum(s string) (string, error) {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" {
		return "", nil
	}
	algo, digest, found := strings.Cut(s, ":")
	if !found {
		digest = algo
		switch len(digest) {
		case 32:
			algo = "md5"
		case 40:
			algo = "sha1"
		case 64:
			algo = "sha256"
		case 128:
			algo = "sha512"
		default:
			return "", fmt.Errorf("unrecognized checksum length")
		}
	}
	if newHash(algo) == nil {
		return "", fmt.Errorf("unsupported checksum %q (use md5, sha1, sha256 or sha512)", algo)
	}
	if _, err := hex.DecodeString(digest); err != nil {
		return "", fmt.Errorf("checksum isn't hexadecimal")
	}
	return algo + ":" + digest, nil
}

func newHash(algo string) hash.Hash {
	switch algo {
	case "md5":
		return md5.New()
	case "sha1":
		return sha1.New()
	case "sha256":
		return sha256.New()
	case "sha512":
		return sha512.New()
	}
	return nil
}

// checkSum hashes path and compares it with want ("algo:hex").
func checkSum(ctx context.Context, path, want string, progress func(done, total int64)) (bool, string, error) {
	algo, digest, _ := strings.Cut(want, ":")
	h := newHash(algo)
	if h == nil {
		return false, "", fmt.Errorf("unsupported checksum %q", algo)
	}
	f, err := os.Open(path)
	if err != nil {
		return false, "", err
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return false, "", err
	}
	buf := make([]byte, 1<<20)
	var done int64
	for {
		if err := ctx.Err(); err != nil {
			return false, "", err
		}
		n, err := f.Read(buf)
		h.Write(buf[:n])
		done += int64(n)
		progress(done, st.Size())
		if err == io.EOF {
			break
		}
		if err != nil {
			return false, "", err
		}
	}
	got := hex.EncodeToString(h.Sum(nil))
	return got == digest, algo + ":" + got, nil
}
