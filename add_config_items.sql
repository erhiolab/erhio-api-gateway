-- 添加配置项到 api_gateway_config 表

-- 网关配置
INSERT INTO api_gateway_config (id, group_name, type, value, title, description, version)
VALUES 
('gateway.local-cache-expire', 'gateway', 'int', '10', '本地缓存过期时间(分钟)', '默认10分钟, 用于存储本地缓存', 1),
('gateway.redis-cache-expire', 'gateway', 'int', '60', 'Redis缓存过期时间(分钟)', '默认60分钟, 用于存储Redis缓存', 1),
('gateway.api-root', 'gateway', 'string', '/_gateway/api', '网关API根路由', '用于访问网关自身的API接口', 1);

-- 认证配置
INSERT INTO api_gateway_config (id, group_name, type, value, title, description, version)
VALUES 
('auth.master-key', 'auth', 'string', 'b8757330c18fe65a8ce7b0733355e292', '加密密钥', '用于加密和解密请求体的密钥', 1),
('auth.timestamp-window', 'auth', 'int', '5', '时间戳窗口(秒)', '默认5秒, 用于校验请求时间戳是否在5秒内', 1),
('auth.nonce-window', 'auth', 'int', '2', 'nonce 唯一性校验窗口(次)', '默认2次, 用于校验请求nonce是否在2次内', 1),
('auth.nonce-window-hour', 'auth', 'int', '24', 'nonce 唯一性校验窗口(小时)', '默认24小时, 用于校验请求nonce是否在24小时内', 1),
('auth.qps-limit', 'auth', 'int', '3', '每秒最大请求数', '默认3次, 用于限制每秒请求数', 1),
('auth.qpm-limit', 'auth', 'int', '100', '每分钟最大请求数', '默认100次, 用于限制每分钟请求数', 1),
('auth.ip-black-list', 'auth', 'string', '', '全局IP黑名单', '用于限制访问的IP地址, 格式为IP地址列表, 每行一个IP地址', 1),
('auth.country-black-list', 'auth', 'string', '', '全局国家黑名单', '用于限制访问的国家, 格式为国家列表, 每一行一个国家', 1);
