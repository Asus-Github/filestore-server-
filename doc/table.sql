-- 文件云盘数据库表结构 (MySQL 5.7+ / 8.0)
-- 使用: mysql -uroot -p < doc/table.sql

CREATE DATABASE IF NOT EXISTS `fileserver` DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
USE `fileserver`;

-- 创建文件表(按文件内容sha1去重的唯一文件)
CREATE TABLE IF NOT EXISTS `tbl_file` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `file_sha1` char(40) NOT NULL DEFAULT '' COMMENT '文件hash',
  `file_name` varchar(256) NOT NULL DEFAULT '' COMMENT '首次上传时的文件名',
  `file_size` bigint NOT NULL DEFAULT '0' COMMENT '文件大小',
  `file_addr` varchar(1024) NOT NULL DEFAULT '' COMMENT '文件存储位置',
  `create_at` datetime DEFAULT CURRENT_TIMESTAMP COMMENT '创建日期',
  `update_at` datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新日期',
  `status` int NOT NULL DEFAULT '0' COMMENT '状态(1可用/2禁用/3已删除)',
  `ext1` int DEFAULT '0' COMMENT '备用字段1',
  `ext2` text COMMENT '备用字段2',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_file_hash` (`file_sha1`),
  KEY `idx_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 创建用户表
CREATE TABLE IF NOT EXISTS `tbl_user` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `user_name` varchar(64) NOT NULL DEFAULT '' COMMENT '用户名',
  `user_pwd` varchar(256) NOT NULL DEFAULT '' COMMENT '用户加密后的密码(bcrypt)',
  `email` varchar(64) NOT NULL DEFAULT '' COMMENT '邮箱',
  `phone` varchar(128) NOT NULL DEFAULT '' COMMENT '手机号',
  `email_validated` tinyint(1) DEFAULT 0 COMMENT '邮箱是否已验证',
  `phone_validated` tinyint(1) DEFAULT 0 COMMENT '手机号是否已验证',
  `signup_at` datetime DEFAULT CURRENT_TIMESTAMP COMMENT '注册日期',
  `last_active` datetime DEFAULT CURRENT_TIMESTAMP COMMENT '最后登录时间',
  `profile` text COMMENT '用户属性',
  `status` int NOT NULL DEFAULT '0' COMMENT '账户状态(0正常/1禁用/2锁定/3标记删除)',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_username` (`user_name`),
  KEY `idx_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 创建用户token表(一个用户可以同时在多处登录, 只保存token的sha256)
CREATE TABLE IF NOT EXISTS `tbl_user_token` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `user_name` varchar(64) NOT NULL DEFAULT '' COMMENT '用户名',
  `token_hash` char(64) NOT NULL DEFAULT '' COMMENT '登录token的sha256',
  `create_at` datetime DEFAULT CURRENT_TIMESTAMP COMMENT '登录时间',
  `expire_at` datetime NOT NULL COMMENT '过期时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_token` (`token_hash`),
  KEY `idx_user_expire` (`user_name`, `expire_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 创建用户文件表
CREATE TABLE IF NOT EXISTS `tbl_user_file` (
  `id` bigint NOT NULL PRIMARY KEY AUTO_INCREMENT,
  `user_name` varchar(64) NOT NULL,
  `file_sha1` char(40) NOT NULL DEFAULT '' COMMENT '文件hash',
  `file_size` bigint NOT NULL DEFAULT '0' COMMENT '文件大小',
  `file_name` varchar(256) NOT NULL DEFAULT '' COMMENT '文件名',
  `upload_at` datetime DEFAULT CURRENT_TIMESTAMP COMMENT '上传时间',
  `last_update` datetime DEFAULT CURRENT_TIMESTAMP
          ON UPDATE CURRENT_TIMESTAMP COMMENT '最后修改时间',
  `status` int NOT NULL DEFAULT '0' COMMENT '文件状态(0正常1已删除2禁用)',
  UNIQUE KEY `idx_user_file` (`user_name`, `file_sha1`),
  KEY `idx_user_status_upload` (`user_name`, `status`, `upload_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
