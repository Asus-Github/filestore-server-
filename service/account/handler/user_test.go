package handler

import (
	"strings"
	"testing"

	"filestore-server/util"
)

func TestGenToken(t *testing.T) {
	a, err := GenToken()
	if err != nil || !IsTokenFormatValid(a) {
		t.Fatalf("GenToken: %q %v", a, err)
	}
	b, _ := GenToken()
	if a == b {
		t.Error("tokens should be unique")
	}
	for _, bad := range []string{"", strings.Repeat("a", 40), strings.Repeat("A", 64), strings.Repeat("g", 64)} {
		if IsTokenFormatValid(bad) {
			t.Errorf("%q should be invalid", bad)
		}
	}
	if HashToken(a) == a || len(HashToken(a)) != 64 {
		t.Error("HashToken should return sha256 hex")
	}
}

func TestCheckPassword(t *testing.T) {
	enc, err := encryptPassword("secret123")
	if err != nil {
		t.Fatal(err)
	}
	if ok, upgrade := checkPassword(enc, "secret123"); !ok || upgrade {
		t.Errorf("bcrypt: ok=%v upgrade=%v", ok, upgrade)
	}
	if ok, _ := checkPassword(enc, "wrong"); ok {
		t.Error("wrong password accepted")
	}
	// 兼容旧版sha1+盐的密码, 并提示升级
	legacy := legacyEncrypt("oldpass")
	if len(legacy) != 40 || legacy != util.Sha1([]byte("oldpass*#890")) {
		t.Fatalf("legacy hash changed: %s", legacy)
	}
	if ok, upgrade := checkPassword(legacy, "oldpass"); !ok || !upgrade {
		t.Errorf("legacy: ok=%v upgrade=%v", ok, upgrade)
	}
	if ok, _ := checkPassword(legacy, "wrong"); ok {
		t.Error("legacy wrong password accepted")
	}
}

func TestValidPassword(t *testing.T) {
	for _, ok := range []string{"123456", "密码密码密码", strings.Repeat("a", 64)} {
		if !validPassword(ok) {
			t.Errorf("%q should be valid", ok)
		}
	}
	// 超过bcrypt的72字节限制(24个中文=72字节可以, 25个不行)
	for _, bad := range []string{"", "12345", strings.Repeat("a", 65), strings.Repeat("密", 25)} {
		if validPassword(bad) {
			t.Errorf("%q should be invalid", bad)
		}
	}
}
