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

CREATE TABLE review_info (
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
`content` varchar(512) NOT NULL COMMENT '评价内容',
`pic_info` varchar(1024) NOT NULL DEFAULT '' COMMENT '媒体信息：图⽚',
`video_info` varchar(1024) NOT NULL DEFAULT '' COMMENT '媒体信息：视频',
`ext_json` varchar(1024) NOT NULL DEFAULT '' COMMENT '信息扩展',
`ctrl_json` varchar(1024) NOT NULL DEFAULT '' COMMENT '控制扩展',
PRIMARY KEY (`id`),
KEY `idx_delete_at` (`delete_at`) COMMENT '逻辑删除索引',
UNIQUE KEY `uk_reply_id` (`reply_id`) COMMENT '回复id索引',
KEY `idx_review_id` (`review_id`) COMMENT '评价id索引',
KEY `idx_store_id` (`store_id`) COMMENT '店铺id索引'
)ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='评价商家回复表';


CREATE TABLE review_appeal_info (
`id` bigint(32) unsigned NOT NULL AUTO_INCREMENT COMMENT '主键',
`create_by` varchar(48) NOT NULL DEFAULT '' COMMENT '创建⽅标识',
`update_by` varchar(48) NOT NULL DEFAULT '' COMMENT '更新⽅标识',
`create_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
`update_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE
CURRENT_TIMESTAMP COMMENT '更新时间',
`delete_at` timestamp COMMENT '逻辑删除标记',
`version` int(10) unsigned NOT NULL DEFAULT '0' COMMENT '乐观锁标记',
`appeal_id` bigint(32) NOT NULL DEFAULT '0' COMMENT '回复id',
`review_id` bigint(32) NOT NULL DEFAULT '0' COMMENT '评价id',
`store_id` bigint(32) NOT NULL DEFAULT '0' COMMENT '店铺id',
`status` tinyint(4) NOT NULL DEFAULT '10' COMMENT '状态:10待审核；20申诉通过；30申诉驳回',
`reason` varchar(255) NOT NULL COMMENT '申诉原因类别',
`content` varchar(255) NOT NULL COMMENT '申诉内容描述',
`pic_info` varchar(1024) NOT NULL DEFAULT '' COMMENT '媒体信息：图⽚',
`video_info` varchar(1024) NOT NULL DEFAULT '' COMMENT '媒体信息：视频',
`op_remarks` varchar(512) NOT NULL DEFAULT '' COMMENT '运营备注',
`op_user` varchar(64) NOT NULL DEFAULT '' COMMENT '运营者标识',
`ext_json` varchar(1024) NOT NULL DEFAULT '' COMMENT '信息扩展',
`ctrl_json` varchar(1024) NOT NULL DEFAULT '' COMMENT '控制扩展',
PRIMARY KEY (`id`),
KEY `idx_delete_at` (`delete_at`) COMMENT '逻辑删除索引',
KEY `idx_appeal_id` (`appeal_id`) COMMENT '申诉id索引',
UNIQUE KEY `uk_review_id` (`review_id`) COMMENT '评价id索引',
KEY `idx_store_id` (`store_id`) COMMENT '店铺id索引'
)ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='评价商家申诉表';
