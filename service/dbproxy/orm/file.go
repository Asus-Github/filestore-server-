package orm

import (
	"database/sql"
	"log"

	mydb "filestore-server/service/dbproxy/conn"
)

// OnFileUploadFinished : 文件上传完成，保存meta
// 同一hash的文件已存在时更新其存储地址并恢复为可用状态(例如之前被禁用或本地文件丢失后重新上传)
func OnFileUploadFinished(filehash string, filename string,
	filesize int64, fileaddr string) (res ExecResult) {
	_, err := mydb.DBConn().Exec(
		"insert into tbl_file (`file_sha1`,`file_name`,`file_size`,`file_addr`,`status`) "+
			"values (?,?,?,?,?) on duplicate key update "+
			"`file_size`=values(`file_size`),`file_addr`=values(`file_addr`),`status`=values(`status`)",
		filehash, filename, filesize, fileaddr, FileStatusAvailable)
	if err != nil {
		log.Println("Failed to save file meta, err:" + err.Error())
		return errResult(err)
	}
	return okResult(nil)
}

// GetFileMeta : 从mysql获取文件元信息
func GetFileMeta(filehash string) (res ExecResult) {
	tfile := TableFile{}
	err := mydb.DBConn().QueryRow(
		"select file_sha1,file_addr,file_name,file_size from tbl_file "+
			"where file_sha1=? and status=? limit 1", filehash, FileStatusAvailable).Scan(
		&tfile.FileHash, &tfile.FileAddr, &tfile.FileName, &tfile.FileSize)
	if err == sql.ErrNoRows {
		return notFoundResult("文件不存在")
	} else if err != nil {
		log.Println(err.Error())
		return errResult(err)
	}
	return okResult(tfile)
}

// GetFileMetaList : 从mysql批量获取文件元信息
func GetFileMetaList(limit int64) (res ExecResult) {
	rows, err := mydb.DBConn().Query(
		"select file_sha1,file_addr,file_name,file_size from tbl_file "+
			"where status=? order by id desc limit ?", FileStatusAvailable, limit)
	if err != nil {
		log.Println(err.Error())
		return errResult(err)
	}
	defer rows.Close()

	tfiles := []TableFile{}
	for rows.Next() {
		tfile := TableFile{}
		if err = rows.Scan(&tfile.FileHash, &tfile.FileAddr,
			&tfile.FileName, &tfile.FileSize); err != nil {
			log.Println(err.Error())
			return errResult(err)
		}
		tfiles = append(tfiles, tfile)
	}
	if err = rows.Err(); err != nil {
		return errResult(err)
	}
	return okResult(tfiles)
}

// UpdateFileLocation : 更新文件的存储地址(如文件被转移了)
func UpdateFileLocation(filehash string, fileaddr string) (res ExecResult) {
	ret, err := mydb.DBConn().Exec(
		"update tbl_file set `file_addr`=? where `file_sha1`=? limit 1", fileaddr, filehash)
	if err != nil {
		log.Println("更新文件location失败, err:" + err.Error())
		return errResult(err)
	}
	if rf, err := ret.RowsAffected(); err != nil {
		return errResult(err)
	} else if rf <= 0 {
		log.Printf("更新文件location失败, 无匹配记录, filehash:%s", filehash)
		return notFoundResult("无记录更新")
	}
	return okResult(nil)
}
