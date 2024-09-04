package api

import (
	"log"
	"net/http"

	"github.com/AethoceSora/DevContainer/src/internal/config"
	"github.com/gorilla/mux"
)

func NewRouter() *mux.Router {
	r := mux.NewRouter()

	// 定义路由和处理函数
	r.HandleFunc("/start", StartContainerHandler).Methods("GET")
	r.HandleFunc("/stop", StopContainerHandler).Methods("POST")
	r.HandleFunc("/list", ListContainersHandler).Methods("GET")

	return r
}

func StartServer(cfg *config.Config) {
	router := NewRouter()

	// 打印正在监听的端口
	log.Printf("Server is starting and listening on %s", cfg.Listen)

	// 启动HTTP服务器
	if err := http.ListenAndServe(cfg.Listen, router); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
