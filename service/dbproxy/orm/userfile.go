package orm

import (
	"database/sql"
	"log"

	mydb "filestore-server/service/dbproxy/conn"
)

const userFileColumns = "file_sha1,file_name,file_size," +
	"date_format(upload_at,'%Y-%m-%d %H:%i:%s'),date_format(last_update,'%Y-%m-%d %H:%i:%s')"

type rowScanner interface {
	Scan(dest ...interface{}) error
}

func scanUserFile(row rowScanner, username string) (TableUserFile, error) {
	ufile := TableUserFile{UserName: username}
	var uploadAt, lastUpdated sql.NullString
	err := row.Scan(&ufile.FileHash, &ufile.FileName, &ufile.FileSize, &uploadAt, &lastUpdated)
	ufile.UploadAt = uploadAt.String
	ufile.LastUpdated = lastUpdated.String
	return ufile, err
}

// OnUserFileUploadFinished : 更新用户文件表
// 用户再次上传(或秒传)同一文件时: 恢复已删除的记录, 并以最新的文件名/上传时间为准
func OnUserFileUploadFinished(username, filehash, filename string, filesize int64) (res ExecResult) {
	_, err := mydb.DBConn().Exec(
		"insert into tbl_user_file (`user_name`,`file_sha1`,`file_name`,`file_size`,`upload_at`,`status`) "+
			"values (?,?,?,?,NOW(),?) on duplicate key update "+
			"`file_name`=values(`file_name`),`file_size`=values(`file_size`),"+
			"`upload_at`=NOW(),`status`=values(`status`)",
		username, filehash, filename, filesize, UserFileStatusNormal)
	if err != nil {
		log.Println(err.Error())
		return errResult(err)
	}
	return okResult(nil)
}

// QueryUserFileMetas : 分页获取用户文件信息(按上传时间倒序)
func QueryUserFileMetas(username string, limit int64, offset int64) (res ExecResult) {
	conn := mydb.DBConn()
	page := UserFilePage{Files: []TableUserFile{}}
	if err := conn.QueryRow(
		"select count(1) from tbl_user_file where user_name=? and status=?",
		username, UserFileStatusNormal).Scan(&page.Total); err != nil {
		log.Println(err.Error())
		return errResult(err)
	}

	rows, err := conn.Query(
		"select "+userFileColumns+" from tbl_user_file where user_name=? and status=? "+
			"order by upload_at desc, id desc limit ? offset ?",
		username, UserFileStatusNormal, limit, offset)
	if err != nil {
		log.Println(err.Error())
		return errResult(err)
	}
	defer rows.Close()

	for rows.Next() {
		ufile, err := scanUserFile(rows, username)
		if err != nil {
			log.Println(err.Error())
			return errResult(err)
		}
		page.Files = append(page.Files, ufile)
	}
	if err = rows.Err(); err != nil {
		return errResult(err)
	}
	return okResult(page)
}

// DeleteUserFile : 删除文件(标记删除)
func DeleteUserFile(username, filehash string) (res ExecResult) {
	ret, err := mydb.DBConn().Exec(
		"update tbl_user_file set status=? where user_name=? and file_sha1=? and status=? limit 1",
		UserFileStatusDeleted, username, filehash, UserFileStatusNormal)
	if err != nil {
		log.Println(err.Error())
		return errResult(err)
	}
	if n, err := ret.RowsAffected(); err != nil {
		return errResult(err)
	} else if n <= 0 {
		return notFoundResult("文件不存在")
	}
	return okResult(nil)
}

// RenameFileName : 文件重命名, 成功时返回更新后的文件信息
func RenameFileName(username, filehash, filename string) (res ExecResult) {
	ret, err := mydb.DBConn().Exec(
		"update tbl_user_file set file_name=? where user_name=? and file_sha1=? and status=? limit 1",
		filename, username, filehash, UserFileStatusNormal)
	if err != nil {
		log.Println(err.Error())
		return errResult(err)
	}
	if n, err := ret.RowsAffected(); err != nil {
		return errResult(err)
	} else if n <= 0 {
		return notFoundResult("文件不存在")
	}
	return QueryUserFileMeta(username, filehash)
}

// QueryUserFileMeta : 获取用户单个文件信息
func QueryUserFileMeta(username string, filehash string) (res ExecResult) {
	row := mydb.DBConn().QueryRow(
		"select "+userFileColumns+" from tbl_user_file "+
			"where user_name=? and file_sha1=? and status=? limit 1",
		username, filehash, UserFileStatusNormal)
	ufile, err := scanUserFile(row, username)
	if err == sql.ErrNoRows {
		return notFoundResult("文件不存在")
	} else if err != nil {
		log.Println(err.Error())
		return errResult(err)
	}
	return okResult(ufile)
}
