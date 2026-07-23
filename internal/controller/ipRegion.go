package controller

import (
	"elake-api-gateway/internal/app"
	"elake-api-gateway/internal/logger"
	"elake-api-gateway/internal/models"
	"elake-api-gateway/internal/utils"
	"net/http"

	"go.uber.org/zap"
)

// IPRegion 获取 IP 所属地
func IPRegion(app *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := utils.NewQueryParser(r)
		ip := q.StringOpt("ip", utils.GetClientIP(r))
		rec, err := app.IPDB.GetAll(ip)
		if err != nil {
			logger.Log.Error("IP解析失败",
				zap.String("ip", ip),
				zap.Error(err),
			)
			utils.InternalServerError(w, "解析IP失败")
			return
		}
		if rec == nil {
			logger.Log.Warn("IP解析未找到IP信息",
				zap.String("ip", ip),
			)
			utils.BadRequest(w, "客户端IP为空")
			return
		}
		var location = &models.IPLocation{
			IP:           ip,
			CountryShort: rec.CountryShort,
			CountryLong:  rec.CountryLong,
			Region:       rec.Region,
			City:         rec.City,
			Latitude:     rec.Latitude,
			Longitude:    rec.Longitude,
			Zipcode:      rec.Zipcode,
			Timezone:     rec.Timezone,
		}
		utils.Success(w, location)
	}
}
