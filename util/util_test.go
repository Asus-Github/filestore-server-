package util

import (
	"strconv"
	"strings"
	"testing"
	"time"
)

func itoa(n int64) string { return strconv.FormatInt(n, 10) }

func TestSanitizeFileName(t *testing.T) {
	cases := []struct {
		in, want string
		ok       bool
	}{
		{"report.pdf", "report.pdf", true},
		{"  空格 文件.txt  ", "空格 文件.txt", true},
		{"../../etc/passwd", "passwd", true},
		{`C:\Users\a\b.txt`, "b.txt", true},
		{`<img src=x>"q".bin`, `<img src=x>"q".bin`, true},
		{"", "", false},
		{"   ", "", false},
		{"..", "", false},
		{"dir/", "", false},
		{"a\x00b", "", false},
		{"a\nb", "", false},
		{strings.Repeat("文", 255), strings.Repeat("文", 255), true},
		{strings.Repeat("文", 256), "", false},
		{string([]byte{0xff, 0xfe}), "", false},
	}
	for _, c := range cases {
		got, err := SanitizeFileName(c.in)
		if (err == nil) != c.ok || got != c.want {
			t.Errorf("SanitizeFileName(%q) = %q, %v; want %q ok=%v", c.in, got, err, c.want, c.ok)
		}
	}
}

func TestValidateNewFileName(t *testing.T) {
	for _, bad := range []string{"a/b", `a\b`, "", " ", "../x"} {
		if _, err := ValidateNewFileName(bad); err == nil {
			t.Errorf("ValidateNewFileName(%q) should fail", bad)
		}
	}
	if got, err := ValidateNewFileName("  新名字 v2.bin "); err != nil || got != "新名字 v2.bin" {
		t.Errorf("got %q, %v", got, err)
	}
}

func TestIsValidUsername(t *testing.T) {
	for _, ok := range []string{"abc", "user_01", "a.b-c", "测试用户", strings.Repeat("a", 32)} {
		if !IsValidUsername(ok) {
			t.Errorf("%q should be valid", ok)
		}
	}
	for _, bad := range []string{"ab", "bad name", "<script>", "a/b", strings.Repeat("a", 33), ""} {
		if IsValidUsername(bad) {
			t.Errorf("%q should be invalid", bad)
		}
	}
}

func TestIsValidSha1(t *testing.T) {
	if !IsValidSha1(Sha1([]byte("x"))) {
		t.Error("sha1 of x should be valid")
	}
	for _, bad := range []string{"", "abc", strings.Repeat("A", 40), strings.Repeat("g", 40), strings.Repeat("a", 41)} {
		if IsValidSha1(bad) {
			t.Errorf("%q should be invalid", bad)
		}
	}
}

func TestSha1Stream(t *testing.T) {
	s := &Sha1Stream{}
	if s.Sum() != "da39a3ee5e6b4b0d3255bfef95601890afd80709" {
		t.Error("empty stream sum mismatch")
	}
	s = &Sha1Stream{}
	s.Update([]byte("hello "))
	s.Update([]byte("world"))
	if s.Sum() != Sha1([]byte("hello world")) {
		t.Error("stream sum mismatch")
	}
}

func TestContentDisposition(t *testing.T) {
	got := ContentDisposition("attachment", `报告 "v2".pdf`)
	want := `attachment; filename="__ _v2_.pdf"; filename*=UTF-8''%E6%8A%A5%E5%91%8A%20%22v2%22.pdf`
	if got != want {
		t.Errorf("got  %s\nwant %s", got, want)
	}
	if got := ContentDisposition("inline", "a b.txt"); got != `inline; filename="a b.txt"; filename*=UTF-8''a%20b.txt` {
		t.Errorf("got %s", got)
	}
	// 非ASCII全部替换后不应为空
	if got := asciiFallback("中文"); got != "__" {
		t.Errorf("got %q", got)
	}
}

func TestDownloadSign(t *testing.T) {
	exp := time.Now().Add(time.Minute).Unix()
	sign := SignDownload("secret", "alice", "abc", exp)
	ok, expired := VerifyDownloadSign("secret", "alice", "abc", itoa(exp), sign)
	if !ok || expired {
		t.Fatalf("valid sign rejected: ok=%v expired=%v", ok, expired)
	}
	for _, c := range []struct{ secret, user, hash, exp, sign string }{
		{"other", "alice", "abc", itoa(exp), sign},
		{"secret", "bob", "abc", itoa(exp), sign},
		{"secret", "alice", "abd", itoa(exp), sign},
		{"secret", "alice", "abc", itoa(exp + 1), sign},
		{"secret", "alice", "abc", "notanumber", sign},
		{"secret", "alice", "abc", itoa(exp), ""},
	} {
		if ok, _ := VerifyDownloadSign(c.secret, c.user, c.hash, c.exp, c.sign); ok {
			t.Errorf("tampered sign accepted: %+v", c)
		}
	}
	past := time.Now().Add(-time.Minute).Unix()
	ok, expired = VerifyDownloadSign("secret", "alice", "abc", itoa(past), SignDownload("secret", "alice", "abc", past))
	if ok || !expired {
		t.Errorf("expired sign: ok=%v expired=%v", ok, expired)
	}
}

func TestRandomHex(t *testing.T) {
	a, err := RandomHex(32)
	if err != nil || len(a) != 64 {
		t.Fatalf("RandomHex: %q %v", a, err)
	}
	b, _ := RandomHex(32)
	if a == b {
		t.Error("RandomHex should not repeat")
	}
}
