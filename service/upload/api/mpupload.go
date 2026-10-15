package api

import (
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gomodule/redigo/redis"

	rPool "filestore-server/cache/redis"
	cmn "filestore-server/common"
	cfg "filestore-server/config"
	"filestore-server/middleware"
	dbcli "filestore-server/service/dbproxy/client"
	"filestore-server/util"
)

// MultipartUploadInfo : 初始化信息
type MultipartUploadInfo struct {
	FileHash   string
	FileSize   int64
	UploadID   string
	ChunkSize  int64
	ChunkCount int
	// ChunkExists : 已上传完成的分块序号(从1开始), 用于断点续传
	ChunkExists []int
}

const (
	keyPrefixUpload = "MP_"      // 分块上传会话信息(hash)
	keyPrefixResume = "MP_IDX_"  // 用户+文件hash -> uploadID, 用于断点续传
	keyPrefixLock   = "MP_LOCK_" // 合并分块时的互斥锁
	fieldChunkIdx   = "chkidx_"
)

var uploadIDRegexp = regexp.MustCompile(`^[0-9a-f]{32}$`)

func uploadKey(uploadID string) string { return keyPrefixUpload + uploadID }

func resumeKey(username, filehash string, filesize int64) string {
	return fmt.Sprintf("%s%s_%s_%d", keyPrefixResume, username, filehash, filesize)
}

func partDir(uploadID string) string { return filepath.Join(cfg.TempPartRootDir, uploadID) }

func partPath(uploadID string, index int) string {
	return filepath.Join(partDir(uploadID), strconv.Itoa(index))
}

// chunkCountOf : 根据文件大小计算分块数量
func chunkCountOf(filesize, chunkSize int64) int {
	return int((filesize + chunkSize - 1) / chunkSize)
}

// expectedChunkSize : 第index(从1开始)个分块应有的大小
func expectedChunkSize(info *uploadSession, index int) int64 {
	if index < info.ChunkCount {
		return info.ChunkSize
	}
	return info.FileSize - int64(info.ChunkCount-1)*info.ChunkSize
}

// uploadSession : 存放在redis中的分块上传会话
type uploadSession struct {
	UploadID   string
	Username   string
	FileHash   string
	FileName   string
	FileSize   int64
	ChunkSize  int64
	ChunkCount int
	Done       map[int]bool
}

// loadSession : 从redis读取分块上传会话, 不存在时返回(nil, nil)
func loadSession(conn redis.Conn, uploadID string) (*uploadSession, error) {
	values, err := redis.StringMap(conn.Do("HGETALL", uploadKey(uploadID)))
	if err != nil {
		return nil, err
	}
	if len(values) == 0 {
		return nil, nil
	}
	s := &uploadSession{
		UploadID: uploadID,
		Username: values["username"],
		FileHash: values["filehash"],
		FileName: values["filename"],
		Done:     map[int]bool{},
	}
	s.FileSize, _ = strconv.ParseInt(values["filesize"], 10, 64)
	s.ChunkSize, _ = strconv.ParseInt(values["chunksize"], 10, 64)
	s.ChunkCount, _ = strconv.Atoi(values["chunkcount"])
	for k, v := range values {
		if !strings.HasPrefix(k, fieldChunkIdx) || v != "1" {
			continue
		}
		if idx, err := strconv.Atoi(strings.TrimPrefix(k, fieldChunkIdx)); err == nil {
			s.Done[idx] = true
		}
	}
	if s.ChunkSize <= 0 || s.ChunkCount <= 0 || s.FileSize <= 0 {
		return nil, errors.New("invalid upload session: " + uploadID)
	}
	return s, nil
}

// doneChunks : 已完成的分块序号(升序), 同时校验分块文件确实存在且大小正确
func (s *uploadSession) doneChunks() []int {
	list := []int{}
	for idx := range s.Done {
		if idx < 1 || idx > s.ChunkCount {
			continue
		}
		if fi, err := os.Stat(partPath(s.UploadID, idx)); err != nil || fi.Size() != expectedChunkSize(s, idx) {
			continue
		}
		list = append(list, idx)
	}
	sort.Ints(list)
	return list
}

func (s *uploadSession) toInfo() MultipartUploadInfo {
	return MultipartUploadInfo{
		FileHash:    s.FileHash,
		FileSize:    s.FileSize,
		UploadID:    s.UploadID,
		ChunkSize:   s.ChunkSize,
		ChunkCount:  s.ChunkCount,
		ChunkExists: s.doneChunks(),
	}
}

// getSession : 读取并校验会话归属当前用户, 失败时已输出响应
func getSession(c *gin.Context, conn redis.Conn, uploadID string) (*uploadSession, bool) {
	if !uploadIDRegexp.MatchString(uploadID) {
		middleware.Fail(c, cmn.StatusParamInvalid, "uploadid参数无效")
		return nil, false
	}
	s, err := loadSession(conn, uploadID)
	if err != nil {
		log.Println(err)
		middleware.Fail(c, cmn.StatusServerError, "服务错误")
		return nil, false
	}
	if s == nil || s.Username != middleware.Username(c) {
		middleware.Fail(c, cmn.StatusParamInvalid, "上传任务不存在或已过期, 请重新上传")
		return nil, false
	}
	return s, true
}

// InitialMultipartUploadHandler : 初始化分块上传
// 参数: filehash, filesize, filename; 同一用户对同一文件重复初始化时返回已有会话以支持断点续传
func InitialMultipartUploadHandler(c *gin.Context) {
	// 1. 解析用户请求参数
	username := middleware.Username(c)
	filehash := strings.ToLower(strings.TrimSpace(c.PostForm("filehash")))
	filesize, err := strconv.ParseInt(c.PostForm("filesize"), 10, 64)
	if !util.IsValidSha1(filehash) || err != nil || filesize <= 0 {
		middleware.Fail(c, cmn.StatusParamInvalid, "参数无效")
		return
	}
	filename, err := util.SanitizeFileName(c.PostForm("filename"))
	if err != nil {
		middleware.Fail(c, cmn.StatusParamInvalid, err.Error())
		return
	}

	// 2. 获得redis的一个连接
	rConn := rPool.RedisPool().Get()
	defer rConn.Close()

	// 3. 已存在未完成的上传任务则直接复用(断点续传)
	rKey := resumeKey(username, filehash, filesize)
	if oldID, err := redis.String(rConn.Do("GET", rKey)); err == nil && uploadIDRegexp.MatchString(oldID) {
		if s, err := loadSession(rConn, oldID); err == nil && s != nil && s.Username == username {
			if filename != s.FileName {
				rConn.Do("HSET", uploadKey(oldID), "filename", filename)
			}
			touchSession(rConn, oldID, rKey)
			middleware.OK(c, "OK", s.toInfo())
			return
		}
	}

	// 4. 生成分块上传的初始化信息
	uploadID, err := util.RandomHex(16)
	if err != nil {
		middleware.Fail(c, cmn.StatusServerError, "服务错误")
		return
	}
	s := &uploadSession{
		UploadID:   uploadID,
		Username:   username,
		FileHash:   filehash,
		FileName:   filename,
		FileSize:   filesize,
		ChunkSize:  cfg.ChunkSize,
		ChunkCount: chunkCountOf(filesize, cfg.ChunkSize),
		Done:       map[int]bool{},
	}

	// 5. 将初始化信息写入到redis缓存, 并设置过期时间
	_, err = rConn.Do("HSET", uploadKey(uploadID),
		"username", username,
		"filehash", filehash,
		"filename", filename,
		"filesize", filesize,
		"chunksize", s.ChunkSize,
		"chunkcount", s.ChunkCount)
	if err == nil {
		_, err = rConn.Do("SET", rKey, uploadID, "EX", int(cfg.MultipartUploadTTL.Seconds()))
	}
	if err != nil {
		log.Println(err)
		middleware.Fail(c, cmn.StatusServerError, "服务错误")
		return
	}
	touchSession(rConn, uploadID, rKey)

	// 6. 将响应初始化数据返回到客户端
	middleware.OK(c, "OK", s.toInfo())
}

// touchSession : 刷新会话的过期时间
// 同时刷新分块目录的修改时间, 避免会话仍有效时分块目录被 CleanExpiredParts 误删
func touchSession(conn redis.Conn, uploadID, rKey string) {
	ttl := int(cfg.MultipartUploadTTL.Seconds())
	conn.Do("EXPIRE", uploadKey(uploadID), ttl)
	if rKey != "" {
		conn.Do("EXPIRE", rKey, ttl)
	}
	now := time.Now()
	_ = os.Chtimes(partDir(uploadID), now, now)
}

// UploadPartHandler : 上传文件分块
// 参数(query): uploadid, index(从1开始); 请求体为分块的二进制内容
func UploadPartHandler(c *gin.Context) {
	// 1. 解析用户请求参数
	uploadID := c.Query("uploadid")
	index, err := strconv.Atoi(c.Query("index"))
	if err != nil {
		middleware.Fail(c, cmn.StatusParamInvalid, "index参数无效")
		return
	}

	// 2. 获得redis连接池中的一个连接
	rConn := rPool.RedisPool().Get()
	defer rConn.Close()

	s, ok := getSession(c, rConn, uploadID)
	if !ok {
		return
	}
	if index < 1 || index > s.ChunkCount {
		middleware.Fail(c, cmn.StatusParamInvalid, "index超出范围")
		return
	}
	expected := expectedChunkSize(s, index)

	// 3. 先写入临时文件, 校验通过后再重命名, 避免并发/中断导致分块内容不完整
	if err := os.MkdirAll(partDir(uploadID), 0755); err != nil {
		log.Println(err)
		middleware.Fail(c, cmn.StatusServerError, "服务错误")
		return
	}
	tmp, err := os.CreateTemp(partDir(uploadID), ".part-*")
	if err != nil {
		log.Println(err)
		middleware.Fail(c, cmn.StatusServerError, "服务错误")
		return
	}
	h := sha1.New()
	body := http.MaxBytesReader(c.Writer, c.Request.Body, expected+1)
	n, err := io.Copy(io.MultiWriter(tmp, h), body)
	if cerr := tmp.Close(); err == nil {
		err = cerr
	}
	if err == nil && n != expected {
		err = fmt.Errorf("分块大小不正确, 期望%d字节, 实际%d字节", expected, n)
	}
	if err == nil {
		// 可选: 校验分块的sha1
		if chk := strings.ToLower(c.Query("chkhash")); chk != "" && chk != hex.EncodeToString(h.Sum(nil)) {
			err = errors.New("分块校验失败, 内容与chkhash不一致")
		}
	}
	if err == nil {
		err = os.Rename(tmp.Name(), partPath(uploadID, index))
	}
	if err != nil {
		os.Remove(tmp.Name())
		log.Printf("upload part failed, uploadid:%s index:%d err:%v", uploadID, index, err)
		middleware.Fail(c, cmn.StatusUploadFailed, "分块上传失败: "+err.Error())
		return
	}

	// 4. 更新redis缓存状态
	if _, err := rConn.Do("HSET", uploadKey(uploadID), fieldChunkIdx+strconv.Itoa(index), 1); err != nil {
		log.Println(err)
		middleware.Fail(c, cmn.StatusServerError, "服务错误")
		return
	}
	touchSession(rConn, uploadID, resumeKey(s.Username, s.FileHash, s.FileSize))

	// 5. 返回处理结果到客户端
	middleware.OK(c, "OK", gin.H{"Index": index, "Size": n})
}

// CompleteUploadHandler : 通知上传合并
// 参数: uploadid, filehash, filesize, filename(可选, 默认使用初始化时的文件名)
func CompleteUploadHandler(c *gin.Context) {
	// 1. 解析请求参数
	uploadID := c.PostForm("uploadid")

	// 2. 获得redis连接池中的一个连接
	rConn := rPool.RedisPool().Get()
	defer rConn.Close()

	s, ok := getSession(c, rConn, uploadID)
	if !ok {
		return
	}
	if fh := strings.ToLower(strings.TrimSpace(c.PostForm("filehash"))); fh != "" && fh != s.FileHash {
		middleware.Fail(c, cmn.StatusParamInvalid, "filehash与上传任务不一致")
		return
	}
	if fs := c.PostForm("filesize"); fs != "" && fs != strconv.FormatInt(s.FileSize, 10) {
		middleware.Fail(c, cmn.StatusParamInvalid, "filesize与上传任务不一致")
		return
	}
	filename := s.FileName
	if name := c.PostForm("filename"); name != "" {
		var err error
		if filename, err = util.SanitizeFileName(name); err != nil {
			middleware.Fail(c, cmn.StatusParamInvalid, err.Error())
			return
		}
	}

	// 3. 防止重复提交导致并发合并
	lockKey := keyPrefixLock + uploadID
	if locked, err := redis.String(rConn.Do("SET", lockKey, "1", "NX", "EX", 600)); err != nil || locked != "OK" {
		middleware.Fail(c, cmn.StatusUploadFailed, "文件正在合并中, 请稍候")
		return
	}
	defer rConn.Do("DEL", lockKey)

	// 4. 判断是否所有分块上传完成
	if done := s.doneChunks(); len(done) != s.ChunkCount {
		middleware.Resp(c, http.StatusOK, cmn.StatusChunkIncomplete, "分块不完整", gin.H{
			"ChunkCount":  s.ChunkCount,
			"ChunkExists": done,
		})
		return
	}

	// 5. 按序合并分块并校验hash
	tmpPath, err := mergeParts(s)
	if err != nil {
		log.Printf("merge parts failed, uploadid:%s err:%v", uploadID, err)
		if errors.Is(err, errHashMismatch) {
			// 内容有误, 清理本次上传任务, 需要重新上传
			cleanupSession(rConn, s)
			middleware.Fail(c, cmn.StatusFileVerifyFailed, "文件校验失败, 请重新上传")
			return
		}
		middleware.Fail(c, cmn.StatusServerError, "合并失败")
		return
	}
	location, err := commitLocalFile(tmpPath, s.FileHash, s.FileSize)
	if err != nil {
		log.Println(err)
		middleware.Fail(c, cmn.StatusServerError, "合并失败")
		return
	}

	// 6. 更新唯一文件表及用户文件表, 成功后清理分块
	fmeta := dbcli.FileMeta{
		FileSha1: s.FileHash,
		FileName: filename,
		FileSize: s.FileSize,
		Location: location,
	}
	if finishUpload(c, s.Username, fmeta) {
		cleanupSession(rConn, s)
	}
}

var errHashMismatch = errors.New("merged file hash mismatch")

// mergeParts : 将分块按序号合并为一个临时文件, 并校验大小及sha1
func mergeParts(s *uploadSession) (string, error) {
	tmp, err := createTemp(cfg.TempLocalRootDir, ".merge-*")
	if err != nil {
		return "", err
	}
	h := sha1.New()
	w := io.MultiWriter(tmp, h)
	var total int64
	for idx := 1; idx <= s.ChunkCount; idx++ {
		var n int64
		n, err = appendFile(w, partPath(s.UploadID, idx))
		if err != nil {
			break
		}
		total += n
	}
	if cerr := tmp.Close(); err == nil {
		err = cerr
	}
	if err == nil && (total != s.FileSize || hex.EncodeToString(h.Sum(nil)) != s.FileHash) {
		err = errHashMismatch
	}
	if err != nil {
		os.Remove(tmp.Name())
		return "", err
	}
	return tmp.Name(), nil
}

func appendFile(w io.Writer, path string) (int64, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	return io.Copy(w, f)
}

// cleanupSession : 删除分块文件及redis中的会话信息
func cleanupSession(conn redis.Conn, s *uploadSession) {
	conn.Do("DEL", uploadKey(s.UploadID), resumeKey(s.Username, s.FileHash, s.FileSize))
	if err := os.RemoveAll(partDir(s.UploadID)); err != nil {
		log.Println(err)
	}
}

// CancelUploadHandler : 取消分块上传, 清理已上传的分块
func CancelUploadHandler(c *gin.Context) {
	rConn := rPool.RedisPool().Get()
	defer rConn.Close()

	s, ok := getSession(c, rConn, c.PostForm("uploadid"))
	if !ok {
		return
	}
	cleanupSession(rConn, s)
	middleware.OK(c, "已取消", nil)
}

// CleanExpiredParts : 定期清理已过期上传任务遗留的分块目录
func CleanExpiredParts() {
	ttl := cfg.MultipartUploadTTL + time.Hour
	for {
		entries, err := os.ReadDir(cfg.TempPartRootDir)
		if err == nil {
			for _, e := range entries {
				if !e.IsDir() || !uploadIDRegexp.MatchString(e.Name()) {
					continue
				}
				if fi, err := e.Info(); err == nil && time.Since(fi.ModTime()) > ttl {
					log.Println("remove expired upload parts:", e.Name())
					os.RemoveAll(filepath.Join(cfg.TempPartRootDir, e.Name()))
				}
			}
		}
		time.Sleep(time.Hour)
	}
}
