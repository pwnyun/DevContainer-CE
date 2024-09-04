package api

import (
	"fmt"
	"net/http"
	"strconv"

	"encoding/json"
	"log"
	"strings"

	"io"

	"github.com/AethoceSora/DevContainer/src/internal/auth"
	"github.com/AethoceSora/DevContainer/src/internal/config"
	"github.com/AethoceSora/DevContainer/src/internal/k8s"
	"github.com/rs/xid"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// generateSecretToken generates a unique secret token using xid library.
func generateSecretToken() string {
	return xid.New().String()
}

func StartContainerHandler(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromToken(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// 加载配置
	cfg, err := config.Load()
	if err != nil {
		http.Error(w, "Failed to load config", http.StatusInternalServerError)
		return
	}

	// 获取用户当前已经创建的容器列表
	clientset, err := k8s.GetK8sClient()
	if err != nil {
		http.Error(w, "Failed to get Kubernetes client", http.StatusInternalServerError)
		return
	}

	pods, err := clientset.CoreV1().Pods("default").List(r.Context(), metav1.ListOptions{
		LabelSelector: "app=openvscode-server",
	})
	if err != nil {
		http.Error(w, "Failed to list pods", http.StatusInternalServerError)
		return
	}

	// 查找最小的可用容器编号
	usedIndices := make(map[int]bool)
	for _, pod := range pods.Items {
		if strings.HasPrefix(pod.Name, "vscs-"+userID+"-") {
			parts := strings.Split(pod.Name, "-")
			if len(parts) == 3 {
				index, err := strconv.Atoi(parts[2])
				if err == nil {
					usedIndices[index] = true
				}
			}
		}
	}

	var containerIndex int
	for i := 1; i <= cfg.MaxContainers; i++ {
		if !usedIndices[i] {
			containerIndex = i
			break
		}
	}

	// 如果所有编号都已被占用，则无法创建新容器
	if containerIndex == 0 {
		http.Error(w, "Maximum number of containers reached", http.StatusBadRequest)
		return
	}

	// 生成新的容器索引
	containerIndexStr := fmt.Sprintf("%d", containerIndex)
	secretToken := generateSecretToken() // 生成一个新的secretToken

	// 创建新的容器
	err = k8s.StartContainer(userID, containerIndexStr, cfg.ImageName, secretToken)
	if err != nil {
		http.Error(w, "Failed to start container", http.StatusInternalServerError)
		return
	}

	// 创建与容器对应的服务，并获取其NodePort
	service, err := k8s.CreateService(userID, containerIndexStr)
	if err != nil {
		http.Error(w, "Failed to create service", http.StatusInternalServerError)
		return
	}

	// 构建VSCode的URL
	url := strings.ReplaceAll(cfg.ContainerURL, "{port}", service.NodePort)
	url = strings.ReplaceAll(url, "{token}", secretToken)

	log.Printf("Generated connection-token: %s", secretToken)

	// 返回URL给客户端
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf(`{"url": "%s"}`, url)))
}

type StopContainerRequest struct {
	ContainerIndex string `json:"containerIndex"`
}

func StopContainerHandler(w http.ResponseWriter, r *http.Request) {
	var request StopContainerRequest

	// 解析JSON请求体
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid JSON request", http.StatusBadRequest)
		return
	}

	// 从请求头中的JWT获取userID
	userID, err := auth.GetUserIDFromToken(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// 销毁对应的容器
	err = k8s.StopContainer(userID, request.ContainerIndex)
	if err != nil {
		log.Printf("Failed to stop container: %v", err)
		http.Error(w, "Failed to stop container", http.StatusInternalServerError)
		return
	}

	// 销毁对应的服务
	err = k8s.DeleteService(userID, request.ContainerIndex)
	if err != nil {
		log.Printf("Failed to delete service: %v", err)
		http.Error(w, "Failed to delete service", http.StatusInternalServerError)
		return
	}

	// 返回成功消息
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Container and service stopped successfully"))
}

func ListContainersHandler(w http.ResponseWriter, r *http.Request) {
	// 从请求头中的JWT获取userID
	userID, err := auth.GetUserIDFromToken(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	clientset, err := k8s.GetK8sClient()
	if err != nil {
		http.Error(w, "Failed to get Kubernetes client", http.StatusInternalServerError)
		return
	}

	// 列出所有属于该用户的Pod
	pods, err := clientset.CoreV1().Pods("default").List(r.Context(), metav1.ListOptions{
		LabelSelector: "app=openvscode-server",
	})
	if err != nil {
		http.Error(w, "Failed to list pods", http.StatusInternalServerError)
		return
	}

	var userContainers []map[string]interface{}
	for _, pod := range pods.Items {
		// 检查pod的名称是否与该用户的容器匹配
		if strings.HasPrefix(pod.Name, "vscs-"+userID+"-") {
			containerInfo := map[string]interface{}{
				"containerName": pod.Name,
				"creationTime":  pod.CreationTimestamp,
				"status":        pod.Status.Phase,
			}
			userContainers = append(userContainers, containerInfo)
		}
	}

	// 将结果编码为JSON并返回
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(userContainers); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

func GitHubLoginHandler(w http.ResponseWriter, r *http.Request) {
	url := auth.GetGitHubLoginURL()
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

// 处理 GitHub OAuth 回调
func GitHubCallbackHandler(w http.ResponseWriter, r *http.Request) {
	state := r.FormValue("state")
	code := r.FormValue("code")

	// 使用 auth 包中的 ExchangeCodeForToken 函数来获取 token
	token, err := auth.ExchangeCodeForToken(code, state)
	if err != nil {
		log.Printf("Code exchange failed: %s", err.Error())
		http.Error(w, "Code exchange failed", http.StatusInternalServerError)
		return
	}

	// 假设 userID 是通过某种方式识别的，这里简化处理
	userID := "example-user"
	auth.SaveToken(userID, token)

	// 告知用户登录成功
	w.Write([]byte("Login successful, now you can list your repositories at /repos"))
}

func ListReposHandler(w http.ResponseWriter, r *http.Request) {
	// 从请求头中的JWT获取userID
	userID, err := auth.GetUserIDFromToken(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// 从内存中获取 token
	token, ok := auth.GetToken(userID)
	if !ok {
		http.Error(w, "Token not found", http.StatusUnauthorized)
		return
	}

	// 使用 token 获取用户的仓库列表
	client := auth.GithubOauthConfig.Client(r.Context(), token)
	resp, err := client.Get("https://api.github.com/user/repos")
	if err != nil {
		http.Error(w, "Failed to get repositories", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	// 将结果编码为JSON并返回
	w.Header().Set("Content-Type", "application/json")
	if _, err := io.Copy(w, resp.Body); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}
