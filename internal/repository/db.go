package repository

import (
	"context"
	"database/sql"
	"elake-api-gateway/internal/config"
	"elake-api-gateway/internal/logger"
	"elake-api-gateway/internal/models"
	"errors"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

// DBManager 封装数据库操作
type DBManager struct {
	db *sqlx.DB
}

// NewDBManager 初始化管理器
func NewDBManager(db *sqlx.DB) *DBManager {
	return &DBManager{db: db}
}

// GetApiKeyInfo 根据密钥ID获取API密钥信息
func (db *DBManager) GetApiKeyInfo(secretID string) (*models.APIKeyInfo, bool, error) {
	cfg := config.Get()
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.DB.ReadTimeout)*time.Second)
	defer cancel()
	var apiKey models.APIKeyInfo
	query := `SELECT id, user_id, secret_id, secret_key, service_id,
       				qps, qpm, ip_filter_type, ip_list, country_filter_type,
       				country_list, enabled, banned, expires_at
			  FROM api_keys
			  WHERE secret_id = ?`
	err := db.db.GetContext(ctx, &apiKey, query, secretID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, false, nil
		}
		logger.Log.Error("查询API密钥信息错误", zap.String("secretID", secretID), zap.Error(err))
		return nil, false, err
	}
	// 获取关联的路由ID
	routeIDs, err := db.GetApiKeyRouteIDsWithCtx(ctx, apiKey.ID)
	if err != nil {
		return nil, false, err
	}
	apiKey.RouteIDs = routeIDs
	// 解析IP列表和国家列表
	if apiKey.RawIPList != nil && *apiKey.RawIPList != "" {
		apiKey.IPList = strings.Split(strings.TrimSpace(*apiKey.RawIPList), "\n")
	} else {
		apiKey.IPList = []string{}
	}
	if apiKey.RawCountryList != nil && *apiKey.RawCountryList != "" {
		apiKey.CountryList = strings.Split(strings.TrimSpace(*apiKey.RawCountryList), "\n")
	} else {
		apiKey.CountryList = []string{}
	}
	return &apiKey, true, nil
}

// GetApiKeyRouteIDsWithCtx 根据密钥ID获取其所有允许访问的路由ID
func (db *DBManager) GetApiKeyRouteIDsWithCtx(ctx context.Context, keyID int64) ([]int64, error) {
	var routeIDs []int64
	query := `SELECT route_id FROM api_key_routes WHERE key_id = ?`
	err := db.db.SelectContext(ctx, &routeIDs, query, keyID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return []int64{}, nil
		}
		logger.Log.Error("查询API密钥关联路由错误", zap.Int64("keyID", keyID), zap.Error(err))
		return nil, err
	}
	return routeIDs, nil
}

// GetRouteByPathAndMethod [未使用]根据路径和方法精确匹配路由
func (db *DBManager) GetRouteByPathAndMethod(path, method string) (*config.Route, bool, error) {
	cfg := config.Get()
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.DB.ReadTimeout)*time.Second)
	defer cancel()
	var route config.Route
	query := `SELECT id, path, method, service_id, require_auth, require_limit 
              FROM routes
              WHERE path = ? AND method = ? LIMIT 1`
	err := db.db.GetContext(ctx, &route, query, path, method)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, false, nil
		}
		return nil, false, err
	}
	return &route, true, nil
}

// GetServiceWithNodes [未使用]获取服务及其所有可用节点
func (db *DBManager) GetServiceWithNodes(serviceID int64) (*config.Service, bool, error) {
	cfg := config.Get()
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.DB.ReadTimeout)*time.Second)
	defer cancel()
	// 获取服务基本信息
	var service config.Service
	err := db.db.GetContext(ctx, &service, "SELECT id, name FROM services WHERE id = ?", serviceID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, false, nil
		}
		return nil, false, err
	}
	// 获取该服务下所有状态正常且未删除的节点
	var nodes []config.ServiceNode
	queryNodes := `SELECT id, service_id, node_url, weight, status 
                   FROM service_nodes 
                   WHERE service_id = ? AND status = 1 AND is_deleted = FALSE`
	err = db.db.SelectContext(ctx, &nodes, queryNodes, serviceID)
	if err != nil {
		return nil, false, err
	}
	service.Nodes = nodes
	return &service, true, nil
}

// GetAllServices 获取所有服务
func (db *DBManager) GetAllServices() ([]config.Service, error) {
	cfg := config.Get()
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.DB.ReadTimeout)*time.Second)
	defer cancel()
	// 获取所有服务
	var dbServices []config.Service
	query := `SELECT id, name, base_path FROM services`
	err := db.db.SelectContext(ctx, &dbServices, query)
	if err != nil {
		return nil, err
	}
	// 转换为 config.Service 格式
	services := make([]config.Service, 0, len(dbServices))
	for _, dbService := range dbServices {
		// 获取服务节点
		var nodes []config.ServiceNode
		nodeQuery := `SELECT id, service_id, node_url, weight, status
					  FROM service_nodes
					  WHERE service_id = ? AND status = 1 AND is_deleted = FALSE`
		err := db.db.SelectContext(ctx, &nodes, nodeQuery, dbService.ID)
		if err != nil {
			return nil, err
		}
		services = append(services, config.Service{
			ID:       dbService.ID,
			Name:     dbService.Name,
			BasePath: dbService.BasePath,
			Nodes:    nodes,
		})
	}
	return services, nil
}

// GetAllRoutes 获取所有路由
func (db *DBManager) GetAllRoutes() ([]config.Route, error) {
	cfg := config.Get()
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.DB.ReadTimeout)*time.Second)
	defer cancel()
	// 获取所有路由
	var dbRoutes []config.Route
	query := `SELECT r.id, r.path, r.method, r.service_id, s.name as service_name, r.require_auth, r.require_limit, r.ip_limit, r.country_limit, r.qps, r.qpm, r.enabled
              FROM routes r 
              JOIN services s ON r.service_id = s.id`
	err := db.db.SelectContext(ctx, &dbRoutes, query)
	if err != nil {
		return nil, err
	}
	// 转换为 config.Route 格式
	routes := make([]config.Route, 0, len(dbRoutes))
	for _, dbRoute := range dbRoutes {
		routes = append(routes, config.Route{
			ID:           dbRoute.ID,
			Path:         dbRoute.Path,
			Method:       dbRoute.Method,
			ServiceID:    dbRoute.ServiceID,
			ServiceName:  dbRoute.ServiceName,
			RequireAuth:  dbRoute.RequireAuth,
			RequireLimit: dbRoute.RequireLimit,
			IpLimit:      dbRoute.IpLimit,
			CountryLimit: dbRoute.CountryLimit,
			QPS:          dbRoute.QPS,
			QPM:          dbRoute.QPM,
			Enabled:      dbRoute.Enabled,
		})
	}
	return routes, nil
}

// Close 关闭数据库连接
func (db *DBManager) Close() error {
	return db.db.Close()
}
