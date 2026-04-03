SET FOREIGN_KEY_CHECKS = 0;

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
	enabled             TINYINT               DEFAULT 1 COMMENT '是否启用: 0关闭, 1开启',
	banned              TINYINT      NOT NULL DEFAULT 0 COMMENT '封禁状态: 0正常, 1永久封禁, 2临时封禁, 3注销中',
	banned_start        TIMESTAMP    NULL     DEFAULT NULL COMMENT '封禁开始时间',
	banned_end          TIMESTAMP    NULL     DEFAULT NULL COMMENT '封禁结束时间',
	expires_at          TIMESTAMP    NULL COMMENT '过期时间',
	created_at          TIMESTAMP             DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
	updated_at          TIMESTAMP             DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
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
	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
	updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间'
);

-- ----------------------------
-- 服务节点表 (service_nodes)
-- ----------------------------
DROP TABLE IF EXISTS `service_nodes`;
CREATE TABLE service_nodes
(
	id         BIGINT PRIMARY KEY AUTO_INCREMENT COMMENT '服务节点 ID',
	service_id BIGINT       NOT NULL COMMENT '服务 ID',
	node_url   VARCHAR(255) NOT NULL COMMENT '服务节点 URL',
	weight     INT       DEFAULT 1 COMMENT '负载均衡权重',
	status     TINYINT   DEFAULT 1 COMMENT '节点状态: 0不可用, 1可用',
	is_deleted BOOLEAN   DEFAULT FALSE COMMENT '软删除标记',
	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
	updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
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
	require_limit BOOLEAN                                                           NOT NULL DEFAULT FALSE COMMENT '是否需要限流',
	ip_limit      BOOLEAN                                                           NOT NULL DEFAULT FALSE COMMENT '是否需要 IP 限流',
	country_limit BOOLEAN                                                           NOT NULL DEFAULT FALSE COMMENT '是否需要国家 限流',
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
