-- 设置MySQL客户端字符集
SET NAMES utf8mb4;
SET CHARACTER SET utf8mb4;
SET character_set_connection=utf8mb4;

-- 创建数据库（如果不存在）
CREATE DATABASE IF NOT EXISTS reviewdb DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
USE reviewdb;

-- 删除已存在的表（重新创建）
DROP TABLE IF EXISTS review_appeal_info;
DROP TABLE IF EXISTS review_reply_info; 
DROP TABLE IF EXISTS review_info;
DROP TABLE IF EXISTS stores;
DROP TABLE IF EXISTS users;

-- 创建用户表
CREATE TABLE IF NOT EXISTS users (
    id BIGINT UNSIGNED NOT NULL PRIMARY KEY,
    username VARCHAR(50) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    role ENUM('customer', 'merchant', 'reviewer') NOT NULL,
    email VARCHAR(100) NOT NULL UNIQUE,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 店铺表
CREATE TABLE IF NOT EXISTS stores (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    store_id BIGINT UNSIGNED NOT NULL UNIQUE,
    user_id BIGINT UNSIGNED NOT NULL,
    name VARCHAR(100) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    KEY `idx_user_id` (`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 评论表
CREATE TABLE IF NOT EXISTS review_info (
  `id` bigint(32) unsigned NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `create_by` varchar(48) NOT NULL DEFAULT '' COMMENT '创建人',
  `update_by` varchar(48) NOT NULL DEFAULT '' COMMENT '更新人',
  `create_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `update_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  `delete_at` timestamp COMMENT '删除时间',
  `version` int(10) unsigned NOT NULL DEFAULT '0' COMMENT '版本号',
  `review_id` bigint(32) NOT NULL DEFAULT '0' COMMENT '评论ID',
  `content` varchar(512) NOT NULL COMMENT '评论内容',
  `score` tinyint(4) NOT NULL DEFAULT '0' COMMENT '评分',
  `service_score` tinyint(4) NOT NULL DEFAULT '0' COMMENT '服务评分',
  `express_score` tinyint(4) NOT NULL DEFAULT '0' COMMENT '快递评分',
  `has_media` tinyint(4) NOT NULL DEFAULT '0' COMMENT '是否有媒体',
  `order_id` bigint(32) NOT NULL DEFAULT '0' COMMENT '订单ID',
  `sku_id` bigint(32) NOT NULL DEFAULT '0' COMMENT 'SKU ID',
  `spu_id` bigint(32) NOT NULL DEFAULT '0' COMMENT 'SPU ID',
  `store_id` bigint(32) NOT NULL DEFAULT '0' COMMENT '店铺ID',
  `user_id` bigint(32) NOT NULL DEFAULT '0' COMMENT '用户ID',
  `anonymous` tinyint(4) NOT NULL DEFAULT '0' COMMENT '是否匿名',
  `tags` varchar(1024) NOT NULL DEFAULT '' COMMENT '标签JSON',
  `pic_info` varchar(1024) NOT NULL DEFAULT '' COMMENT '图片信息',
  `video_info` varchar(1024) NOT NULL DEFAULT '' COMMENT '视频信息',
  `status` tinyint(4) NOT NULL DEFAULT '10' COMMENT '状态',
  `is_default` tinyint(4) NOT NULL DEFAULT '0' COMMENT '是否默认',
  `has_reply` tinyint(4) NOT NULL DEFAULT '0' COMMENT '是否有回复',
  `op_reason` varchar(512) NOT NULL DEFAULT '' COMMENT '操作原因',
  `op_remarks` varchar(512) NOT NULL DEFAULT '' COMMENT '操作备注',
  `op_user` varchar(64) NOT NULL DEFAULT '' COMMENT '操作用户',
  `goods_snapshoot` varchar(2048) NOT NULL DEFAULT '' COMMENT '商品快照',
  `ext_json` varchar(1024) NOT NULL DEFAULT '' COMMENT '扩展JSON',
  `ctrl_json` varchar(1024) NOT NULL DEFAULT '' COMMENT '控制JSON',
  PRIMARY KEY (`id`),
  KEY `idx_delete_at` (`delete_at`) COMMENT '删除时间索引',
  UNIQUE KEY `uk_review_id` (`review_id`) COMMENT '评论ID唯一索引',
  KEY `idx_order_id` (`order_id`) COMMENT '订单ID索引',
  KEY `idx_user_id` (`user_id`) COMMENT '用户ID索引'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='评论信息表';


CREATE TABLE review_reply_info (
`id` bigint(32) unsigned NOT NULL AUTO_INCREMENT COMMENT '主键',
`create_by` varchar(48) NOT NULL DEFAULT '' COMMENT '创建⽅标识',
`update_by` varchar(48) NOT NULL DEFAULT '' COMMENT '更新⽅标识',
`create_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
`update_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE
CURRENT_TIMESTAMP COMMENT '更新时间',
`delete_at` timestamp COMMENT '逻辑删除标记',
`version` int(10) unsigned NOT NULL DEFAULT '0' COMMENT '乐观锁标记',
`reply_id` bigint(32) NOT NULL DEFAULT '0' COMMENT '回复id',
`review_id` bigint(32) NOT NULL DEFAULT '0' COMMENT '评价id',
`store_id` bigint(32) NOT NULL DEFAULT '0' COMMENT '店铺id',
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='回复信息表';







