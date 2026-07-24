SET FOREIGN_KEY_CHECKS = 0;

-- ----------------------------
-- 网关配置 (api_gateway_config)
-- ----------------------------
DROP TABLE IF EXISTS `api_gateway_config`;
CREATE TABLE api_gateway_config
(
	id          VARCHAR(255) PRIMARY KEY COMMENT '配置ID',
	group_name  VARCHAR(64)                                     NOT NULL COMMENT '配置分组',
	type        ENUM ('string', 'int', 'float', 'bool', 'json') NOT NULL COMMENT '类型',
	value       TEXT COMMENT '配置值',
	title       VARCHAR(255) DEFAULT NULL COMMENT '配置标题',
	description VARCHAR(255) DEFAULT NULL COMMENT '配置说明',
	version     INT          DEFAULT 1 COMMENT '版本号',
	updated_at  TIMESTAMP    DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

-- ----------------------------
-- 全局黑名单 (api_gateway_blacklist)
-- ----------------------------
DROP TABLE IF EXISTS `api_gateway_blacklist`;
CREATE TABLE api_gateway_blacklist
(
	id          BIGINT PRIMARY KEY AUTO_INCREMENT COMMENT '黑名单 ID',
	type        ENUM ('ip', 'domain', 'country') NOT NULL COMMENT '黑名单类型',
	value       TEXT COMMENT '黑名单值',
	description VARCHAR(255) DEFAULT NULL COMMENT '黑名单说明',
	created_at  TIMESTAMP    DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

-- ----------------------------
-- 用户表 (users)
-- ----------------------------
DROP TABLE IF EXISTS `users`;
CREATE TABLE `users`
(
	id                BIGINT PRIMARY KEY AUTO_INCREMENT COMMENT '用户 ID',
	email             VARCHAR(100) NOT NULL UNIQUE COMMENT '登录邮箱(主登录方式)',
	username          VARCHAR(32)  NOT NULL COMMENT '用户名(英文/下划线)',
	limit_service_num INT(11)      NOT NULL DEFAULT 5 COMMENT '该用户允许创建的服务最大数量配额',
	banned            TINYINT      NOT NULL DEFAULT 0 COMMENT '封禁状态: 0-正常, 1-永久封禁, 2-临时封禁, 3-注销中',
	banned_start      TIMESTAMP    NULL     DEFAULT NULL COMMENT '封禁开始时间',
	banned_end        TIMESTAMP    NULL     DEFAULT NULL COMMENT '封禁结束时间',
	created_at        TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
	updated_at        TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间'
);

-- ----------------------------
-- 登录记录表 (user_login_records)
-- ----------------------------
DROP TABLE IF EXISTS `user_login_records`;
CREATE TABLE `user_login_records`
(
	id         BIGINT PRIMARY KEY AUTO_INCREMENT COMMENT '登录记录 ID',
	user_id    BIGINT       NOT NULL COMMENT '用户ID',
	ip         VARCHAR(45)  NOT NULL COMMENT '登录IP地址',
	region     VARCHAR(255) NULL     DEFAULT NULL COMMENT '登录地区',
	device     VARCHAR(255) NULL     DEFAULT NULL COMMENT '登录设备',
	created_at TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '登录时间',
	FOREIGN KEY (user_id) REFERENCES users (id)
);

-- ----------------------------
-- API 密钥表 (api_keys)
-- ----------------------------
DROP TABLE IF EXISTS `api_keys`;
CREATE TABLE api_keys
(
	id                  BIGINT PRIMARY KEY AUTO_INCREMENT COMMENT '密钥 ID',
	user_id             BIGINT       NOT NULL COMMENT '所属用户 ID',
	secret_id           VARCHAR(32)  NOT NULL UNIQUE COMMENT 'Key ID',
	secret_key          VARCHAR(255) NOT NULL COMMENT 'Key',
	service_id          BIGINT       NOT NULL COMMENT '服务 ID',
	qps                 INT                   DEFAULT 0 COMMENT '请求每秒限制',
	qpm                 INT                   DEFAULT 0 COMMENT '请求每分钟限制',
	ip_filter_type      TINYINT               DEFAULT 0 COMMENT 'IP 过滤类型: 0关闭, 1白名单, 2黑名单',
	ip_list             TEXT COMMENT 'IP 列表, 每个 IP 一行',
	country_filter_type TINYINT               DEFAULT 0 COMMENT '国家 过滤类型: 0关闭, 1白名单, 2黑名单',
	country_list        TEXT COMMENT '国家 列表, 每个国家一行',
	domain_filter_type  TINYINT               DEFAULT 0 COMMENT '域名 过滤类型: 0关闭, 1白名单, 2黑名单',
	domain_list         TEXT COMMENT '域名 列表, 每个域名一行',
	enabled             TINYINT               DEFAULT 1 COMMENT '是否启用: 0关闭, 1开启',
	banned              TINYINT      NOT NULL DEFAULT 0 COMMENT '封禁状态: 0正常, 1永久封禁, 2临时封禁, 3注销中',
	banned_start        TIMESTAMP    NULL     DEFAULT NULL COMMENT '封禁开始时间',
	banned_end          TIMESTAMP    NULL     DEFAULT NULL COMMENT '封禁结束时间',
	expires_at          TIMESTAMP    NULL COMMENT '过期时间',
	created_at          TIMESTAMP             DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
	updated_at          TIMESTAMP             DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
	FOREIGN KEY (user_id) REFERENCES users (id),
	FOREIGN KEY (service_id) REFERENCES services (id)
);

-- ----------------------------
-- API 密钥路由关联表 (api_key_routes)
-- ----------------------------
DROP TABLE IF EXISTS `api_key_routes`;
CREATE TABLE api_key_routes
(
	id         BIGINT PRIMARY KEY AUTO_INCREMENT COMMENT '关联 ID',
	service_id BIGINT NOT NULL COMMENT '服务 ID',
	key_id     BIGINT NOT NULL COMMENT '密钥 ID',
	route_id   BIGINT NOT NULL COMMENT '路由 ID',
	FOREIGN KEY (service_id) REFERENCES services (id),
	FOREIGN KEY (key_id) REFERENCES api_keys (id),
	FOREIGN KEY (route_id) REFERENCES routes (id),
	UNIQUE KEY uk_key_route (key_id, route_id) COMMENT '唯一索引: 密钥 ID + 路由 ID'
);

-- ----------------------------
-- 服务表 (services)
-- ----------------------------
DROP TABLE IF EXISTS `services`;
CREATE TABLE services
(
	id         BIGINT PRIMARY KEY AUTO_INCREMENT COMMENT '服务 ID',
	name       VARCHAR(100) NOT NULL UNIQUE COMMENT '服务名称',
	base_path  VARCHAR(64)  NOT NULL COMMENT '服务基础路径',
	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
	updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间'
);

-- ----------------------------
-- 服务节点表 (service_nodes)
-- ----------------------------
DROP TABLE IF EXISTS `service_nodes`;
CREATE TABLE service_nodes
(
	id           BIGINT PRIMARY KEY AUTO_INCREMENT COMMENT '服务节点 ID',
	service_id   BIGINT       NOT NULL COMMENT '服务 ID',
	node_url     VARCHAR(255) NOT NULL COMMENT '服务节点 URL',
	weight       INT           DEFAULT 1 COMMENT '负载均衡权重',
	max_conn     INT           DEFAULT 100 COMMENT '最大连接数',
	status       TINYINT       DEFAULT 1 COMMENT '节点状态: 0不可用, 1可用',
	availability DECIMAL(5, 2) DEFAULT 100.00 COMMENT '可用度百分比',
	is_deleted   BOOLEAN       DEFAULT FALSE COMMENT '软删除标记',
	created_at   TIMESTAMP     DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
	updated_at   TIMESTAMP     DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
	FOREIGN KEY (service_id) REFERENCES services (id)
);

-- ----------------------------
-- 路由表 (routes)
-- ----------------------------
DROP TABLE IF EXISTS `routes`;
CREATE TABLE routes
(
	id            BIGINT PRIMARY KEY AUTO_INCREMENT COMMENT '路由 ID',
	path          VARCHAR(255)                                                      NOT NULL COMMENT '路由路径',
	method        ENUM ('GET', 'POST', 'PUT', 'DELETE', 'OPTIONS', 'HEAD', 'PATCH') NOT NULL COMMENT '路由方法',
	service_id    BIGINT                                                            NOT NULL COMMENT '服务 ID',
	require_auth  BOOLEAN                                                           NOT NULL DEFAULT FALSE COMMENT '是否需要认证',
	ip_limit      BOOLEAN                                                           NOT NULL DEFAULT FALSE COMMENT '是否需要 IP 限流',
	country_limit BOOLEAN                                                           NOT NULL DEFAULT FALSE COMMENT '是否需要国家限流',
	domain_limit  BOOLEAN                                                           NOT NULL DEFAULT FALSE COMMENT '是否需要域名限流',
	qps           INT                                                                        DEFAULT 0 COMMENT '请求每秒限制',
	qpm           INT                                                                        DEFAULT 0 COMMENT '请求每分钟限制',
	enabled       TINYINT                                                                    DEFAULT 1 COMMENT '是否启用: 0关闭, 1开启',
	created_at    TIMESTAMP                                                                  DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
	updated_at    TIMESTAMP                                                                  DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
	FOREIGN KEY (service_id) REFERENCES services (id),
	UNIQUE KEY uk_path_method (path, method) COMMENT '唯一索引: 路由路径 + 方法',
	INDEX idx_service_id (service_id) COMMENT '服务 ID 索引',
	INDEX idx_path_method (path, method) COMMENT '路由路径和方法索引'
);

-- ----------------------------
-- 定时任务
-- ----------------------------
DELIMITER $$

-- 自动清理过期封禁
DROP EVENT IF EXISTS `ev_auto_unban_logic`$$
CREATE EVENT `ev_auto_unban_logic`
	ON SCHEDULE
		EVERY 30 SECOND
	ON COMPLETION PRESERVE
	ENABLE
	COMMENT '自动清理过期封禁'
	DO
	BEGIN
		-- 仅解封状态为 2 (临时封禁) 且时间已过期的记录
		UPDATE `api_keys`
		SET `banned`       = 0,
			`banned_start` = NULL,
			`banned_end`   = NULL
		WHERE `banned` = 2
		  AND `banned_end` IS NOT NULL
		  AND `banned_end` < NOW();
	END$$

DELIMITER ;

-- 尝试开启调度器
SET GLOBAL event_scheduler = ON;
SET FOREIGN_KEY_CHECKS = 1;

-- ----------------------------
-- 初始化配置
-- ----------------------------

-- 网关配置
INSERT INTO api_gateway_config (id, group_name, type, value, title, description, version)
VALUES ('gateway.local-cache-expire', 'gateway', 'int', '10', '本地缓存过期时间(分钟)', '默认10分钟, 用于存储本地缓存',
		1),
	   ('gateway.redis-cache-expire', 'gateway', 'int', '60', 'Redis缓存过期时间(分钟)',
		'默认60分钟, 用于存储Redis缓存', 1),
	   ('gateway.api-root', 'gateway', 'string', '/_gateway/api', '网关API根路由',
		'默认/_gateway/api, 用于访问网关自身的API接口', 1),
	   ('gateway.node-timeout', 'gateway', 'int', '3', '节点超时时间(秒)', '默认3秒, 代理到单个节点的超时时间', 1),
	   ('gateway.total-timeout', 'gateway', 'int', '10', '总超时时间(秒)', '默认10秒, 遍历所有节点的总超时限制', 1);

-- Email配置
INSERT INTO api_gateway_config (id, group_name, type, value, title, description, version)
VALUES ('email.host', 'email', 'string', 'smtp.qq.com', 'SMTP主机地址', '用于发送邮件的SMTP主机地址', 1),
	   ('email.port', 'email', 'int', '587', 'SMTP主机端口', '用于发送邮件的SMTP主机端口', 1),
	   ('email.username', 'email', 'string', '2444236088@qq.com', 'SMTP用户名', '用于发送邮件的SMTP用户名/邮箱', 1),
	   ('email.password', 'email', 'string', 'tgaiwyfrofceeacd', 'SMTP密码', '用于发送邮件的SMTP密码', 1),
	   ('email.timeout', 'email', 'int', '10', 'SMTP超时时间(秒)', '用于发送邮件的SMTP超时时间', 1),
	   ('email.max-retry', 'email', 'int', '3', '最大重试次数', '用于发送邮件的最大重试次数', 1);

-- 认证配置
INSERT INTO api_gateway_config (id, group_name, type, value, title, description, version)
VALUES ('auth.master-key', 'auth', 'string', 'b8757330c18fe65a8ce7b0733355e292', '加密密钥',
		'用于加密和解密请求体的密钥', 1),
	   ('auth.timestamp-window', 'auth', 'int', '5', '时间戳窗口(秒)', '默认5秒, 用于校验请求时间戳是否在5秒内', 1),
	   ('auth.nonce-window', 'auth', 'int', '2', 'nonce 唯一性校验窗口(次)', '默认2次, 用于校验请求nonce是否在2次内',
		1),
	   ('auth.nonce-window-hour', 'auth', 'int', '24', 'nonce 唯一性校验窗口(小时)',
		'默认24小时, 用于校验请求nonce是否在24小时内', 1),
	   ('auth.qps-limit', 'auth', 'int', '5', '每秒最大请求数', '默认5次, 用于限制每秒请求数', 1),
	   ('auth.qpm-limit', 'auth', 'int', '150', '每分钟最大请求数', '默认150次, 用于限制每分钟请求数', 1);
