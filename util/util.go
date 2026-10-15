package util

import (
	"crypto/hmac"
	"crypto/md5"
	"crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"hash"
	"io"
	"os"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

type Sha1Stream struct {
	_sha1 hash.Hash
}

func (obj *Sha1Stream) Update(data []byte) {
	if obj._sha1 == nil {
		obj._sha1 = sha1.New()
	}
	obj._sha1.Write(data)
}

func (obj *Sha1Stream) Sum() string {
	if obj._sha1 == nil {
		obj._sha1 = sha1.New()
	}
	return hex.EncodeToString(obj._sha1.Sum(nil))
}

func Sha1(data []byte) string {
	_sha1 := sha1.New()
	_sha1.Write(data)
	return hex.EncodeToString(_sha1.Sum(nil))
}

func FileSha1(file *os.File) string {
	_sha1 := sha1.New()
	io.Copy(_sha1, file)
	return hex.EncodeToString(_sha1.Sum(nil))
}

func Sha256(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func MD5(data []byte) string {
	_md5 := md5.New()
	_md5.Write(data)
	return hex.EncodeToString(_md5.Sum(nil))
}

func FileMD5(file *os.File) string {
	_md5 := md5.New()
	io.Copy(_md5, file)
	return hex.EncodeToString(_md5.Sum(nil))
}

// HmacSha256 : 计算hmac-sha256签名(hex)
func HmacSha256(secret, data string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(data))
	return hex.EncodeToString(mac.Sum(nil))
}

// SecureEqual : 常量时间比较, 防止时序攻击
func SecureEqual(a, b string) bool {
	return hmac.Equal([]byte(a), []byte(b))
}

// RandomHex : 生成n字节随机数的hex字符串(长度2n)
func RandomHex(n int) (string, error) {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func PathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

// GetFileSize : 获取文件大小, 文件不存在时返回-1
func GetFileSize(filename string) int64 {
	fi, err := os.Stat(filename)
	if err != nil {
		return -1
	}
	return fi.Size()
}

var (
	sha1Regexp     = regexp.MustCompile(`^[0-9a-f]{40}$`)
	usernameRegexp = regexp.MustCompile(`^[\p{L}\p{N}_.\-]{3,32}$`)
)

// IsValidSha1 : 是否为合法的sha1 hex字符串(小写)
func IsValidSha1(s string) bool {
	return sha1Regexp.MatchString(s)
}

// IsValidUsername : 用户名3~32位, 仅允许字母(含中文)、数字、下划线、点、中划线
func IsValidUsername(s string) bool {
	return usernameRegexp.MatchString(s)
}

// MaxFileNameLen : 文件名最大字符数(与表结构 varchar(256) 对应)
const MaxFileNameLen = 255

// ErrInvalidFileName : 非法文件名
var ErrInvalidFileName = errors.New("文件名不能为空, 不能超过255个字符, 且不能包含 / \\ 或控制字符")

// SanitizeFileName : 校验并规范化用户提交的文件名
// 去掉首尾空白及客户端可能带上的目录部分, 拒绝控制字符与路径分隔符
func SanitizeFileName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if idx := strings.LastIndexAny(name, `/\`); idx >= 0 {
		name = strings.TrimSpace(name[idx+1:])
	}
	if name == "" || name == "." || name == ".." || !utf8.ValidString(name) ||
		utf8.RuneCountInString(name) > MaxFileNameLen {
		return "", ErrInvalidFileName
	}
	for _, r := range name {
		if unicode.IsControl(r) {
			return "", ErrInvalidFileName
		}
	}
	return name, nil
}

// ValidateNewFileName : 校验重命名时的新文件名(不做截取, 含分隔符直接拒绝)
func ValidateNewFileName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if strings.ContainsAny(name, `/\`) {
		return "", ErrInvalidFileName
	}
	return SanitizeFileName(name)
}
