-- 数据库结构 (整理自原 00001-00032 迁移, 各表为最终结构). 初始数据 (默认分类方案、内置会员组、admin 账号、
-- 语言、播放器等) 由 database/seed 写入, 不在迁移中.
-- users 与 categories 的自增ID从 10000 开始 (后台新增的分类不会与采集站的分类ID冲突).
-- 之后的表结构变更请新增 00002_xxx.sql, 不要修改已执行的迁移.

-- +goose Up
-- ---------------------------------------------------------------- 管理员与会员
CREATE TABLE `users` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `created_at` datetime DEFAULT NULL,
  `updated_at` datetime DEFAULT NULL,
  `deleted_at` datetime DEFAULT NULL,
  `user_name` varchar(255) DEFAULT NULL,
  `password` varchar(255) DEFAULT NULL,
  `salt` varchar(255) DEFAULT NULL,
  `email` varchar(255) DEFAULT NULL,
  `gender` bigint DEFAULT NULL,
  `nick_name` varchar(255) DEFAULT NULL,
  `avatar` varchar(255) DEFAULT NULL,
  `status` bigint DEFAULT NULL,
  `reserve1` varchar(255) DEFAULT NULL,
  `reserve2` varchar(255) DEFAULT NULL,
  `reserve3` varchar(255) DEFAULT NULL,
  `last_login_at` datetime DEFAULT NULL COMMENT '上次登录时间',
  `last_login_ip` varchar(64) NOT NULL DEFAULT '' COMMENT '上次登录IP',
  `login_count` int NOT NULL DEFAULT '0' COMMENT '登录次数',
  `permissions` text COMMENT '后台权限 key, JSON 数组',
  PRIMARY KEY (`id`),
  KEY `idx_users_deleted_at` (`deleted_at`)
) ENGINE=InnoDB AUTO_INCREMENT=10000 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

CREATE TABLE `admin_logs` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) NOT NULL,
  `user_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT '管理员ID, 登录失败时为 0',
  `user_name` varchar(100) NOT NULL DEFAULT '' COMMENT '管理员账号 (登录失败时为输入的账号)',
  `ip` varchar(64) NOT NULL DEFAULT '',
  `method` varchar(10) NOT NULL DEFAULT '',
  `path` varchar(255) NOT NULL DEFAULT '',
  `action` varchar(100) NOT NULL DEFAULT '' COMMENT '操作名称',
  `params` text COMMENT '请求参数 (密码已隐藏, 过长截断)',
  `success` tinyint(1) NOT NULL DEFAULT '1',
  PRIMARY KEY (`id`),
  KEY `idx_admin_logs_created_at` (`created_at`),
  KEY `idx_admin_logs_user_id` (`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

CREATE TABLE `member_groups` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `created_at` datetime DEFAULT NULL,
  `updated_at` datetime DEFAULT NULL,
  `name` varchar(60) NOT NULL,
  `status` tinyint unsigned NOT NULL DEFAULT '1' COMMENT '1 启用 0 停用',
  `price_day` int NOT NULL DEFAULT '0' COMMENT '包天价格 (点数)',
  `price_week` int NOT NULL DEFAULT '0',
  `price_month` int NOT NULL DEFAULT '0',
  `price_year` int NOT NULL DEFAULT '0',
  `remark` varchar(255) NOT NULL DEFAULT '',
  `permissions` text COMMENT '分类权限, JSON: 分类ID → 权限列表',
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

CREATE TABLE `members` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `created_at` datetime DEFAULT NULL,
  `updated_at` datetime DEFAULT NULL,
  `user_name` varchar(60) NOT NULL,
  `password` varchar(64) NOT NULL,
  `salt` varchar(32) NOT NULL,
  `nick_name` varchar(60) NOT NULL DEFAULT '',
  `email` varchar(100) NOT NULL DEFAULT '',
  `group_id` bigint unsigned NOT NULL DEFAULT '2' COMMENT 'member_groups.id',
  `points` int NOT NULL DEFAULT '0' COMMENT '点数',
  `expire_at` datetime DEFAULT NULL COMMENT '会员组到期时间, 为空表示不过期',
  `status` tinyint unsigned NOT NULL DEFAULT '1' COMMENT '1 启用 0 停用',
  `last_login_at` datetime DEFAULT NULL,
  `last_login_ip` varchar(64) NOT NULL DEFAULT '',
  `login_count` int NOT NULL DEFAULT '0',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_members_user_name` (`user_name`),
  KEY `idx_members_group` (`group_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- ---------------------------------------------------------------- 分类方案、分类、语言
CREATE TABLE `category_schemes` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `created_at` datetime DEFAULT NULL,
  `updated_at` datetime DEFAULT NULL,
  `name` varchar(60) NOT NULL,
  `sort` int NOT NULL DEFAULT '0' COMMENT '越小越靠前',
  `default_lang` varchar(16) NOT NULL DEFAULT 'zh-CN',
  `langs` varchar(255) NOT NULL DEFAULT '["zh-CN"]',
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

CREATE TABLE `categories` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `scheme_id` bigint unsigned NOT NULL DEFAULT '1' COMMENT 'category_schemes.id',
  `created_at` datetime DEFAULT NULL,
  `updated_at` datetime DEFAULT NULL,
  `type` tinyint unsigned NOT NULL DEFAULT '1' COMMENT '1视频 2文章 3演员 4网站',
  `parent_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT '0 为顶级分类',
  `name` varchar(60) NOT NULL,
  `slug` varchar(60) NOT NULL COMMENT '前台 URL 标识, 唯一',
  `status` tinyint unsigned NOT NULL DEFAULT '1' COMMENT '1启用 0停用',
  `sort` int NOT NULL DEFAULT '0' COMMENT '越小越靠前',
  `i18n` text,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_categories_scheme_slug` (`scheme_id`,`slug`),
  KEY `idx_categories_tree` (`scheme_id`,`type`,`parent_id`,`sort`)
) ENGINE=InnoDB AUTO_INCREMENT=10000 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

CREATE TABLE `languages` (
  `code` varchar(16) NOT NULL,
  `name` varchar(40) NOT NULL,
  `libre_code` varchar(16) NOT NULL DEFAULT '' COMMENT '翻译服务 (LibreTranslate) 的语言代码',
  `enabled` tinyint(1) NOT NULL DEFAULT '1',
  `sort` int NOT NULL DEFAULT '0',
  `created_at` datetime DEFAULT NULL,
  `updated_at` datetime DEFAULT NULL,
  PRIMARY KEY (`code`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- ---------------------------------------------------------------- 影片
CREATE TABLE `vod` (
  `vod_id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `type_id` bigint NOT NULL DEFAULT '0',
  `type_id_1` bigint NOT NULL DEFAULT '0',
  `type_name` varchar(60) NOT NULL DEFAULT '',
  `vod_name` varchar(255) NOT NULL DEFAULT '',
  `vod_name_key` varchar(255) NOT NULL DEFAULT '' COMMENT '比对用片名 (去空白与标点, 小写)',
  `vod_sub` varchar(255) NOT NULL DEFAULT '',
  `vod_en` varchar(255) NOT NULL DEFAULT '',
  `vod_status` tinyint unsigned NOT NULL DEFAULT '1' COMMENT '审核 1 已审核 0 未审核 (前台不显示)',
  `vod_level` tinyint unsigned NOT NULL DEFAULT '0' COMMENT '推荐 0-9, 0 为未推荐',
  `vod_lock` tinyint unsigned NOT NULL DEFAULT '0' COMMENT '1 锁定 (采集不更新)',
  `vod_letter` varchar(10) NOT NULL DEFAULT '',
  `vod_class` varchar(255) NOT NULL DEFAULT '',
  `vod_pic` varchar(1024) NOT NULL DEFAULT '',
  `vod_actor` varchar(255) NOT NULL DEFAULT '',
  `vod_director` varchar(255) NOT NULL DEFAULT '',
  `vod_writer` varchar(255) NOT NULL DEFAULT '',
  `vod_remarks` varchar(255) NOT NULL DEFAULT '',
  `vod_pubdate` varchar(100) NOT NULL DEFAULT '',
  `vod_area` varchar(255) NOT NULL DEFAULT '',
  `vod_lang` varchar(255) NOT NULL DEFAULT '',
  `vod_year` varchar(10) NOT NULL DEFAULT '',
  `vod_state` varchar(255) NOT NULL DEFAULT '',
  `vod_hits` bigint unsigned NOT NULL DEFAULT '0',
  `vod_score` decimal(3,1) unsigned NOT NULL DEFAULT '0.0',
  `vod_time` int unsigned NOT NULL DEFAULT '0' COMMENT '更新时间',
  `vod_time_add` int unsigned NOT NULL DEFAULT '0' COMMENT '添加时间',
  `vod_douban_id` bigint unsigned NOT NULL DEFAULT '0',
  `vod_douban_score` decimal(3,1) unsigned NOT NULL DEFAULT '0.0',
  `vod_content` mediumtext NOT NULL,
  `vod_play_from` text NOT NULL,
  `vod_play_url` mediumtext NOT NULL,
  `vod_down_from` varchar(255) NOT NULL DEFAULT '',
  `vod_down_url` mediumtext NOT NULL,
  PRIMARY KEY (`vod_id`),
  KEY `idx_vod_type` (`type_id`,`vod_status`,`vod_time`),
  KEY `idx_vod_type1` (`type_id_1`,`vod_status`,`vod_time`),
  KEY `idx_vod_time` (`vod_time`),
  KEY `idx_vod_time_add` (`vod_time_add`),
  KEY `idx_vod_hits` (`vod_hits`),
  KEY `idx_vod_score` (`vod_score`),
  KEY `idx_vod_year` (`vod_year`),
  KEY `idx_vod_name` (`vod_name`),
  KEY `idx_vod_douban` (`vod_douban_id`),
  KEY `idx_vod_name_key` (`vod_name_key`),
  KEY `idx_vod_level` (`vod_level`),
  FULLTEXT KEY `ft_vod_name` (`vod_name`,`vod_sub`) /*!50100 WITH PARSER `ngram` */ ,
  FULLTEXT KEY `ft_vod_class` (`vod_class`) /*!50100 WITH PARSER `ngram` */ 
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

CREATE TABLE `vod_type` (
  `vod_id` bigint unsigned NOT NULL,
  `type_id` bigint NOT NULL COMMENT '分类ID',
  `type_id_1` bigint NOT NULL DEFAULT '0' COMMENT '一级分类ID',
  `scheme_id` bigint unsigned NOT NULL DEFAULT '1' COMMENT '分类所属方案',
  `status` tinyint unsigned NOT NULL DEFAULT '1' COMMENT '1 显示 0 隐藏 (分类被停用)',
  PRIMARY KEY (`vod_id`,`type_id`),
  KEY `idx_vod_type_type` (`type_id`,`status`),
  KEY `idx_vod_type_type1` (`type_id_1`,`status`),
  KEY `idx_vod_type_scheme` (`scheme_id`,`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

CREATE TABLE `vod_origin` (
  `source_id` varchar(64) NOT NULL COMMENT 'collect_sources.id',
  `origin_id` bigint NOT NULL COMMENT '采集站的影片ID',
  `vod_id` bigint unsigned NOT NULL COMMENT '本站影片ID',
  `play_from` varchar(500) NOT NULL DEFAULT '' COMMENT '该采集接口提供的播放组代码, 逗号分隔',
  PRIMARY KEY (`source_id`,`origin_id`),
  KEY `idx_vod_origin_vod` (`vod_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

CREATE TABLE `vod_i18n` (
  `vod_id` bigint unsigned NOT NULL,
  `lang` varchar(16) NOT NULL,
  `name` varchar(255) NOT NULL DEFAULT '',
  `sub` varchar(255) NOT NULL DEFAULT '',
  `content` text,
  `actor` varchar(255) NOT NULL DEFAULT '',
  `director` varchar(255) NOT NULL DEFAULT '',
  `writer` varchar(255) NOT NULL DEFAULT '',
  `remarks` varchar(255) NOT NULL DEFAULT '',
  `area` varchar(255) NOT NULL DEFAULT '',
  `lang_text` varchar(255) NOT NULL DEFAULT '',
  `class` varchar(255) NOT NULL DEFAULT '',
  `is_manual` tinyint NOT NULL DEFAULT '0',
  `updated_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`vod_id`,`lang`),
  KEY `idx_vod_i18n_lang` (`lang`),
  FULLTEXT KEY `ft_vod_i18n_name` (`name`,`sub`) /*!50100 WITH PARSER `ngram` */ 
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

CREATE TABLE `vod_search` (
  `search_key` char(32) CHARACTER SET ascii COLLATE ascii_bin NOT NULL COMMENT '搜索键 (关键词+字段 的 md5)',
  `search_word` varchar(128) NOT NULL COMMENT '搜索关键词',
  `search_field` varchar(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL COMMENT '搜索字段名, 多个以 | 分隔',
  `search_hit_count` bigint unsigned NOT NULL DEFAULT '0' COMMENT '搜索命中次数',
  `search_last_hit_time` int unsigned NOT NULL DEFAULT '0' COMMENT '最近命中时间',
  `search_update_time` int unsigned NOT NULL DEFAULT '0' COMMENT '结果更新时间',
  `search_result_count` int unsigned NOT NULL DEFAULT '0' COMMENT '结果数量',
  `search_result_ids` mediumtext CHARACTER SET ascii COLLATE ascii_bin NOT NULL COMMENT '结果影片ID, 英文逗号分隔',
  PRIMARY KEY (`search_key`),
  KEY `idx_vod_search_hits` (`search_hit_count`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

CREATE TABLE `players` (
  `code` varchar(60) NOT NULL COMMENT '播放组代码',
  `created_at` datetime DEFAULT NULL,
  `updated_at` datetime DEFAULT NULL,
  `name` varchar(60) NOT NULL COMMENT '前台线路名称',
  `status` tinyint unsigned NOT NULL DEFAULT '1' COMMENT '1 启用 0 停用 (前台不显示该线路)',
  `sort` int NOT NULL DEFAULT '0' COMMENT '越小越靠前',
  `remark` varchar(255) NOT NULL DEFAULT '' COMMENT '备注',
  PRIMARY KEY (`code`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- ---------------------------------------------------------------- 采集
CREATE TABLE `collect_sources` (
  `id` varchar(64) NOT NULL,
  `created_at` datetime DEFAULT NULL,
  `updated_at` datetime DEFAULT NULL,
  `name` varchar(60) NOT NULL,
  `uri` varchar(500) NOT NULL COMMENT '接口地址',
  `params` varchar(500) NOT NULL DEFAULT '' COMMENT '附加参数, 如 &ct=1',
  `result_model` tinyint unsigned NOT NULL DEFAULT '0' COMMENT '接口类型 0json 1xml',
  `collect_type` tinyint unsigned NOT NULL DEFAULT '0' COMMENT '资源类型 0视频 1文章 2演员 3角色 4网站',
  `operation` tinyint unsigned NOT NULL DEFAULT '0' COMMENT '数据操作 0新增+更新 1新增 2更新',
  `filter_mode` tinyint unsigned NOT NULL DEFAULT '0' COMMENT '地址过滤 0不过滤 1新增+更新 2新增 3更新',
  `filter_code` varchar(255) NOT NULL DEFAULT '' COMMENT '过滤代码, 逗号分隔',
  `filter_year` varchar(255) NOT NULL DEFAULT '' COMMENT '过滤年份, 逗号分隔',
  `sync_image` tinyint unsigned NOT NULL DEFAULT '0' COMMENT '同步图片 0跟随全局 1开启 2关闭',
  `state` tinyint(1) NOT NULL DEFAULT '0' COMMENT '是否启用',
  `interval` int NOT NULL DEFAULT '0' COMMENT '单次请求间隔 ms',
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

CREATE TABLE `collect_binds` (
  `source_id` varchar(64) NOT NULL COMMENT 'collect_sources.id',
  `type_id` bigint NOT NULL COMMENT '采集站分类ID',
  `type_name` varchar(60) NOT NULL DEFAULT '' COMMENT '采集站分类名称',
  `category_id` bigint unsigned NOT NULL COMMENT '本站分类 categories.id',
  `created_at` datetime DEFAULT NULL,
  `updated_at` datetime DEFAULT NULL,
  PRIMARY KEY (`source_id`,`type_id`,`category_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

CREATE TABLE `collect_logs` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `source_id` varchar(64) NOT NULL,
  `source_name` varchar(60) NOT NULL DEFAULT '',
  `mode` varchar(30) NOT NULL DEFAULT '' COMMENT '采集当天 / 采集本周 / 采集所有 / 按ID采集 ...',
  `trigger` varchar(20) NOT NULL DEFAULT '' COMMENT '手动 / 定时任务 / 续采',
  `hours` int NOT NULL DEFAULT '0' COMMENT '采集时长参数 h (负数为全部), 重试失败页时沿用',
  `state` varchar(20) NOT NULL DEFAULT '' COMMENT 'done / stopped / failed / interrupted',
  `added` int NOT NULL DEFAULT '0',
  `updated` int NOT NULL DEFAULT '0',
  `skipped` int NOT NULL DEFAULT '0',
  `failed_pages` int NOT NULL DEFAULT '0',
  `failed_list` text COMMENT '失败的页 JSON [{typeId, page}]',
  `retried` tinyint(1) NOT NULL DEFAULT '0' COMMENT '失败页是否已重试',
  `message` varchar(500) NOT NULL DEFAULT '',
  `started_at` datetime DEFAULT NULL,
  `finished_at` datetime DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_collect_logs_source` (`source_id`,`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- ---------------------------------------------------------------- 站群、海报、广告、SEO
CREATE TABLE `domains` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `created_at` datetime DEFAULT NULL,
  `updated_at` datetime DEFAULT NULL,
  `deleted_at` datetime DEFAULT NULL,
  `domain` varchar(255) DEFAULT NULL,
  `theme` varchar(255) DEFAULT NULL,
  `site_name` varchar(50) DEFAULT NULL,
  `logo` varchar(500) DEFAULT NULL,
  `seo_title` varchar(100) DEFAULT NULL,
  `keyword` varchar(200) DEFAULT NULL,
  `seo_description` varchar(500) DEFAULT NULL,
  `service_email` varchar(100) DEFAULT NULL,
  `analytics_code` text,
  `legal_info` text,
  `state` tinyint(1) NOT NULL DEFAULT '1' COMMENT '网站状态 1开启 0关闭',
  `scheme_id` bigint unsigned NOT NULL DEFAULT '1' COMMENT '使用的分类方案',
  `i18n` text,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_domains_domain` (`domain`),
  KEY `idx_domains_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

CREATE TABLE `banners` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `scheme_id` bigint unsigned NOT NULL DEFAULT '1',
  `mid` bigint NOT NULL DEFAULT '0' COMMENT '绑定的影片ID (须在该分类方案中), 0 为不链接',
  `name` varchar(255) NOT NULL,
  `poster` varchar(1024) NOT NULL DEFAULT '',
  `picture` varchar(1024) NOT NULL DEFAULT '',
  `sort` int NOT NULL DEFAULT '0',
  `status` tinyint(1) NOT NULL DEFAULT '1',
  `i18n` text COMMENT '各语言的海报名称 JSON {lang: {name}}',
  `created_at` datetime DEFAULT NULL,
  `updated_at` datetime DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_banners_scheme` (`scheme_id`,`sort`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

CREATE TABLE `ads` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `scheme_id` bigint unsigned NOT NULL DEFAULT '1',
  `slot` varchar(30) NOT NULL COMMENT '广告位代码, 如 play_top',
  `name` varchar(100) NOT NULL,
  `image` varchar(1024) NOT NULL,
  `link` varchar(1024) NOT NULL DEFAULT '',
  `sort` int NOT NULL DEFAULT '0',
  `status` tinyint(1) NOT NULL DEFAULT '1',
  `created_at` datetime DEFAULT NULL,
  `updated_at` datetime DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_ads_slot` (`scheme_id`,`slot`,`status`,`sort`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

CREATE TABLE `seo_rules` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `scheme_id` bigint unsigned NOT NULL,
  `page_type` varchar(16) NOT NULL,
  `lang` varchar(16) NOT NULL,
  `title` varchar(255) NOT NULL DEFAULT '',
  `keywords` varchar(255) NOT NULL DEFAULT '',
  `description` varchar(500) NOT NULL DEFAULT '',
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_seo_rules` (`scheme_id`,`page_type`,`lang`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- ---------------------------------------------------------------- 图库与网站地图任务
CREATE TABLE `files` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `created_at` datetime DEFAULT NULL,
  `updated_at` datetime DEFAULT NULL,
  `deleted_at` datetime DEFAULT NULL,
  `link` varchar(255) DEFAULT NULL,
  `uid` bigint DEFAULT NULL,
  `relevance_id` bigint DEFAULT NULL,
  `type` bigint DEFAULT NULL,
  `fid` varchar(255) DEFAULT NULL,
  `original_name` varchar(255) NOT NULL DEFAULT '' COMMENT '上传时的原始档名',
  `file_type` varchar(255) DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_files_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

CREATE TABLE `publish_jobs` (
  `id` varchar(32) NOT NULL,
  `type` varchar(20) NOT NULL COMMENT 'sitemap / indexnow',
  `scheme_id` bigint NOT NULL DEFAULT '0',
  `status` varchar(20) NOT NULL COMMENT 'pending / running / completed / failed',
  `total` bigint NOT NULL DEFAULT '0',
  `processed` bigint NOT NULL DEFAULT '0',
  `error` varchar(1000) NOT NULL DEFAULT '',
  `started_at` datetime DEFAULT NULL,
  `finished_at` datetime DEFAULT NULL,
  `created_at` datetime DEFAULT NULL,
  `updated_at` datetime DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_publish_jobs_created` (`created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- +goose Down
DROP TABLE IF EXISTS `publish_jobs`, `files`, `seo_rules`, `ads`, `banners`, `domains`, `collect_logs`, `collect_binds`, `collect_sources`, `players`, `vod_search`, `vod_i18n`, `vod_origin`, `vod_type`, `vod`, `languages`, `categories`, `category_schemes`, `members`, `member_groups`, `admin_logs`, `users`;
