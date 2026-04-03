package repository

import (
	"elake-api-gateway/internal/models"
	"elake-api-gateway/internal/storage"
)

// IPDBManager 封装 IPDB 数据数据库操作
type IPDBManager struct {
	ipdb *storage.IPDB
}

// NewIPDBManager 初始化管理器
func NewIPDBManager(ipdb *storage.IPDB) *IPDBManager {
	return &IPDBManager{ipdb: ipdb}
}

// GetCountryShort 获取 IP 地址对应的国家简称
func (ipdb *IPDBManager) GetCountryShort(ip string) (string, error) {
	if ip == "" {
		return "", nil
	}
	longitude, err := ipdb.ipdb.Get().Get_country_short(ip)
	if err != nil {
		return "", err
	}
	return longitude.Country_short, nil
}

// GetCountryLong 获取 IP 地址对应的国家名称
func (ipdb *IPDBManager) GetCountryLong(ip string) (string, error) {
	if ip == "" {
		return "", nil
	}
	longitude, err := ipdb.ipdb.Get().Get_country_long(ip)
	if err != nil {
		return "", err
	}
	return longitude.Country_long, nil
}

// GetRegion 获取 IP 地址对应的区域
func (ipdb *IPDBManager) GetRegion(ip string) (string, error) {
	if ip == "" {
		return "", nil
	}
	region, err := ipdb.ipdb.Get().Get_region(ip)
	if err != nil {
		return "", err
	}
	return region.Region, nil
}

// GetCity 获取 IP 地址对应的城市
func (ipdb *IPDBManager) GetCity(ip string) (string, error) {
	if ip == "" {
		return "", nil
	}
	city, err := ipdb.ipdb.Get().Get_city(ip)
	if err != nil {
		return "", err
	}
	return city.City, nil
}

// GetLatitude 获取 IP 地址对应的纬度
func (ipdb *IPDBManager) GetLatitude(ip string) (float32, error) {
	if ip == "" {
		return 0, nil
	}
	latitude, err := ipdb.ipdb.Get().Get_latitude(ip)
	if err != nil {
		return 0, err
	}
	return latitude.Latitude, nil
}

// GetLongitude 获取 IP 地址对应的经度
func (ipdb *IPDBManager) GetLongitude(ip string) (float32, error) {
	if ip == "" {
		return 0, nil
	}
	longitude, err := ipdb.ipdb.Get().Get_longitude(ip)
	if err != nil {
		return 0, err
	}
	return longitude.Longitude, nil
}

// GetZipcode 获取 IP 地址对应的邮政编码
func (ipdb *IPDBManager) GetZipcode(ip string) (string, error) {
	if ip == "" {
		return "", nil
	}
	zipcode, err := ipdb.ipdb.Get().Get_zipcode(ip)
	if err != nil {
		return "", err
	}
	return zipcode.Zipcode, nil
}

// GetTimezone 获取 IP 地址对应的时区
func (ipdb *IPDBManager) GetTimezone(ip string) (string, error) {
	if ip == "" {
		return "", nil
	}
	timezone, err := ipdb.ipdb.Get().Get_timezone(ip)
	if err != nil {
		return "", err
	}
	return timezone.Timezone, nil
}

// GetAll 获取 IP 地址的所有信息
func (ipdb *IPDBManager) GetAll(ip string) (*models.IPLocation, error) {
	if ip == "" {
		return nil, nil
	}
	rec, err := ipdb.ipdb.Get().Get_all(ip)
	if err != nil {
		return nil, err
	}
	var location = &models.IPLocation{
		IP:           ip,
		CountryShort: rec.Country_short,
		CountryLong:  rec.Country_long,
		Region:       rec.Region,
		City:         rec.City,
		Latitude:     rec.Latitude,
		Longitude:    rec.Longitude,
		Zipcode:      rec.Zipcode,
		Timezone:     rec.Timezone,
	}
	return location, nil
}

// Close 关闭数据库连接
func (ipdb *IPDBManager) Close() {
	ipdb.ipdb.Get().Close()
}
