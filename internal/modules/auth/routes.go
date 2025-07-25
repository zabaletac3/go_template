package auth

import (
	"go-template/internal/container"
	"net/http"
)

func RegisterRoutes(deps *container.Dependencies) {

		logger := deps.GetLogger("auth")
		logger.Info("Registering auth module routes")
		
		mux := deps.Mux
		
		mux.HandleFunc("POST /api/v1/auth/login", (func(w http.ResponseWriter, r *http.Request) {
			logger.Info("Login request received")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Login request received"))
		}))

		logger.Info("✅ Auth module routes registered successfully", 
			"endpoints", 1, 
			"base_path", "/api/v1/auth")

}