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
	routeIDs, err := db.getApiKeyRouteIDsWithCtx(ctx, apiKey.ID)
	if err != nil {
		logger.Log.Error("查询API密钥关联路由错误", zap.Int64("keyID", apiKey.ID), zap.Error(err))
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

// getApiKeyRouteIDsWithCtx 根据密钥ID获取其所有允许访问的路由ID
func (db *DBManager) getApiKeyRouteIDsWithCtx(ctx context.Context, keyID int64) ([]int64, error) {
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

// GetRouteByID 根据路由ID获取路由信息
func (db *DBManager) GetRouteByID(routeID int64) (*models.Route, bool, error) {
	cfg := config.Get()
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.DB.ReadTimeout)*time.Second)
	defer cancel()

	var route models.Route
	query := `SELECT r.id, r.path, r.method, r.service_id, s.name as service_name, r.require_auth, r.ip_limit, r.country_limit, r.qps, r.qpm, r.enabled
              FROM routes r 
              JOIN services s ON r.service_id = s.id
              WHERE r.id = ?`
	err := db.db.GetContext(ctx, &route, query, routeID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, false, nil
		}
		logger.Log.Error("查询路由信息错误", zap.Int64("routeID", routeID), zap.Error(err))
		return nil, false, err
	}
	return &route, true, nil
}

// GetServiceWithNodes 获取服务及其所有可用节点
func (db *DBManager) GetServiceWithNodes(serviceID int64) (*models.Service, bool, error) {
	cfg := config.Get()
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.DB.ReadTimeout)*time.Second)
	defer cancel()
	// 获取服务基本信息
	var service models.Service
	err := db.db.GetContext(ctx, &service, "SELECT id, name FROM services WHERE id = ?", serviceID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, false, nil
		}
		logger.Log.Error("查询服务基本信息错误", zap.Int64("serviceID", serviceID), zap.Error(err))
		return nil, false, err
	}
	// 获取该服务下所有状态正常且未删除的节点
	var nodes []models.ServiceNode
	queryNodes := `SELECT id, service_id, node_url, weight, max_conn, status, availability 
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
func (db *DBManager) GetAllServices() ([]models.Service, error) {
	cfg := config.Get()
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.DB.ReadTimeout)*time.Second)
	defer cancel()
	// 获取所有服务
	var dbServices []models.Service
	query := `SELECT id, name, base_path FROM services`
	err := db.db.SelectContext(ctx, &dbServices, query)
	if err != nil {
		logger.Log.Error("查询服务列表错误", zap.Error(err))
		return nil, err
	}
	services := make([]models.Service, 0, len(dbServices))
	for _, dbService := range dbServices {
		// 获取服务节点
		var nodes []models.ServiceNode
		nodeQuery := `SELECT id, service_id, node_url, weight, max_conn, status, availability
				  FROM service_nodes
				  WHERE service_id = ? AND status = 1 AND is_deleted = FALSE`
		err := db.db.SelectContext(ctx, &nodes, nodeQuery, dbService.ID)
		if err != nil {
			logger.Log.Error("查询服务节点错误", zap.Int64("serviceID", dbService.ID), zap.Error(err))
			return nil, err
		}
		services = append(services, models.Service{
			ID:       dbService.ID,
			Name:     dbService.Name,
			BasePath: dbService.BasePath,
			Nodes:    nodes,
		})
	}
	return services, nil
}

// GetAllRoutes 获取所有路由
func (db *DBManager) GetAllRoutes() ([]models.Route, error) {
	cfg := config.Get()
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.DB.ReadTimeout)*time.Second)
	defer cancel()
	var dbRoutes []models.Route
	query := `SELECT r.id, r.path, r.method, r.service_id, s.name as service_name, r.require_auth, r.ip_limit, r.country_limit, r.qps, r.qpm, r.enabled
              FROM routes r 
              JOIN services s ON r.service_id = s.id`
	err := db.db.SelectContext(ctx, &dbRoutes, query)
	if err != nil {
		logger.Log.Error("查询路由列表错误", zap.Error(err))
		return nil, err
	}
	routes := make([]models.Route, 0, len(dbRoutes))
	for _, dbRoute := range dbRoutes {
		routes = append(routes, models.Route{
			ID:           dbRoute.ID,
			Path:         dbRoute.Path,
			Method:       dbRoute.Method,
			ServiceID:    dbRoute.ServiceID,
			ServiceName:  dbRoute.ServiceName,
			RequireAuth:  dbRoute.RequireAuth,
			IpLimit:      dbRoute.IpLimit,
			CountryLimit: dbRoute.CountryLimit,
			QPS:          dbRoute.QPS,
			QPM:          dbRoute.QPM,
			Enabled:      dbRoute.Enabled,
		})
	}
	return routes, nil
}

// GetDashboardStats 获取所有统计信息
func (db *DBManager) GetDashboardStats() (int64, int64, int64, error) {
	cfg := config.Get()
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.DB.ReadTimeout)*time.Second)
	defer cancel()
	var servicesCount, nodesCount, routesCount int64
	query := `
	SELECT 
		(SELECT COUNT(*) FROM services) AS services_count,
		(SELECT COUNT(*) FROM service_nodes WHERE is_deleted = FALSE) AS nodes_count,
		(SELECT COUNT(*) FROM routes) AS routes_count
	`
	err := db.db.QueryRowContext(ctx, query).Scan(&servicesCount, &nodesCount, &routesCount)
	if err != nil {
		logger.Log.Error("统计仪表盘信息错误", zap.Error(err))
		return 0, 0, 0, err
	}
	return servicesCount, nodesCount, routesCount, nil
}

// GetDB 获取数据库连接
func (db *DBManager) GetDB() *sqlx.DB {
	return db.db
}

// Close 关闭数据库连接
func (db *DBManager) Close() error {
	return db.db.Close()
}

// UpdateNodeAvailability 批量更新节点可用度
func (db *DBManager) UpdateNodeAvailability(updates []models.NodeAvailabilityUpdate) error {
	if len(updates) == 0 {
		return nil
	}
	cfg := config.Get()
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.DB.WriteTimeout)*time.Second)
	defer cancel()
	// 构建批量更新SQL
	cases := make([]string, 0, len(updates))
	ids := make([]int64, 0, len(updates))
	args := make([]any, 0, len(updates)*2)
	for _, update := range updates {
		cases = append(cases, "WHEN ? THEN ?")
		ids = append(ids, update.NodeID)
		args = append(args, update.NodeID, update.Availability)
	}
	query := `UPDATE service_nodes SET availability = CASE id ` + strings.Join(cases, " ") + ` END WHERE id IN (?` + strings.Repeat(", ?", len(ids)-1) + `)`
	args = append(args, toAnySlice(ids)...)
	_, err := db.db.ExecContext(ctx, query, args...)
	if err != nil {
		logger.Log.Error("批量更新节点可用度错误", zap.Error(err))
		return err
	}
	return nil
}

// toAnySlice 转换int64切片为any切片
func toAnySlice(ids []int64) []any {
	result := make([]any, len(ids))
	for i, id := range ids {
		result[i] = id
	}
	return result
}
