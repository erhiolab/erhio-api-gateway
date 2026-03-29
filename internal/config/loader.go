package config

// Load 加载配置
func Load() *Config {
	return &Config{
		Routes: []Route{
			{
				Path:    "/api/user",
				Method:  "GET",
				Service: "user-service",
			},
		},
		Services: []Service{
			{
				Name: "user-service",
				Nodes: []string{
					"http://localhost:3000",
				},
			},
		},
	}
}
