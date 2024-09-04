package config

import (
	"os"
)

type Config struct {
	Listen        string
	ContainerURL  string
	ImageName     string
	MaxContainers int // 添加此字段，表示允许的最大容器数量
}

func Load() (*Config, error) {
	return &Config{
		Listen:        getEnv("LISTEN_ADDRESS", "0.0.0.0:30030"),
		ContainerURL:  getEnv("CONTAINER_URL", "http://ctf.qlu.edu.cn:{port}/?tkn={token}"),
		ImageName:     getEnv("IMAGE_NAME", "gitpod/openvscode-server"),
		MaxContainers: 4, // 默认值为 4，可根据需要调整
	}, nil
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
