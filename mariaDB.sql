-- ----------------------------
-- 服务表 (services)
-- ----------------------------
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
CREATE TABLE service_nodes
(
	id         BIGINT PRIMARY KEY AUTO_INCREMENT COMMENT '服务节点 ID',
	service_id BIGINT       NOT NULL COMMENT '服务 ID',
	node_url   VARCHAR(255) NOT NULL COMMENT '服务节点 URL',
	weight     INT       DEFAULT 1 COMMENT '负载均衡权重',
	status     TINYINT   DEFAULT 1 COMMENT '节点状态: 1可用, 0不可用',
	is_deleted BOOLEAN   DEFAULT FALSE COMMENT '软删除标记',
	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
	updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
	FOREIGN KEY (service_id) REFERENCES services (id)
);

-- ----------------------------
-- 路由表 (routes)
-- ----------------------------
CREATE TABLE routes
(
	id            BIGINT PRIMARY KEY AUTO_INCREMENT COMMENT '路由 ID',
	path          VARCHAR(255)                                                      NOT NULL COMMENT '路由路径',
	method        ENUM ('GET', 'POST', 'PUT', 'DELETE', 'OPTIONS', 'HEAD', 'PATCH') NOT NULL COMMENT '路由方法',
	service_id    BIGINT                                                            NOT NULL COMMENT '服务 ID',
	require_auth  BOOLEAN                                                           NOT NULL DEFAULT FALSE COMMENT '是否需要认证',
	require_limit BOOLEAN                                                           NOT NULL DEFAULT FALSE COMMENT '是否需要限流',
	created_at    TIMESTAMP                                                                  DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
	updated_at    TIMESTAMP                                                                  DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
	FOREIGN KEY (service_id) REFERENCES services (id),
	UNIQUE KEY uk_path_method (path, method) COMMENT '防止重复路由',
	INDEX idx_service_id (service_id) COMMENT '服务 ID 索引',
	INDEX idx_path_method (path, method) COMMENT '路由路径和方法索引'
);