package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	ocpp "github.com/yourorg/csms-backend/internal/handler/ocpp"
	"github.com/yourorg/csms-backend/internal/middleware"
	ws "github.com/yourorg/csms-backend/internal/websocket"
)

func NewRouter(hub *ws.Hub) *gin.Engine {
	r := gin.Default()
	r.Use(middleware.CORS())
	r.Use(middleware.Logger())

	// Serve firmware files for OTA update
	r.Static("/firmware", "/tmp")

	// Catch-all for unknown paths — returns clean JSON 404.
	// Prevents scanner probes (/wiki, /login.asp, /) from filling
	// event logs with 404 noise.
	r.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
	})

	api := r.Group("/api")

	// ── Public ──────────────────────────────────────────────────────────────
	api.POST("/auth/login", Login)

	// ── Authenticated ────────────────────────────────────────────────────────
	auth := api.Group("/")
	auth.Use(middleware.JWT())

	auth.POST("/auth/logout", Logout)
	auth.GET("/auth/me", Me)

	// Events WebSocket
	auth.GET("/events/ws", hub.ServeWS)
	// Events history query
	auth.GET("/events/logs", GetEventLogs)

	// ── Devices ──────────────────────────────────────────────────────────────
	dev := auth.Group("/devices")
	{
		dev.GET("", ListDevices)
		dev.GET("/:id", GetDeviceHandler)
		dev.POST("", middleware.RequireCSAdminOrCPOP(), CreateDevice)
		dev.PUT("/:id", UpdateDevice)
		dev.DELETE("/:id", middleware.RequireCSAdminOrCPOP(), DeleteDevice)
	}

	// ── OCPP ─────────────────────────────────────────────────────────────────
	ocppGroup := auth.Group("/ocpp/:deviceId")
	{
		// Configuration
		ocppGroup.GET("/configuration", ocpp.GetConfiguration)
		ocppGroup.POST("/configuration/get", ocpp.GetConfigurationKeys)
		ocppGroup.POST("/configuration/set", ocpp.SetConfiguration)

		// Transactions
		ocppGroup.GET("/transactions", ocpp.GetTransactions)
		ocppGroup.GET("/transactions/active", ocpp.GetActiveTransactions)
		ocppGroup.POST("/remote-start", ocpp.RemoteStart)
		ocppGroup.POST("/remote-stop", ocpp.RemoteStop)

		// Maintenance
		ocppGroup.POST("/reset", ocpp.Reset)
		ocppGroup.POST("/get-log", ocpp.GetLog)
		ocppGroup.POST("/firmware-update", ocpp.FirmwareUpdate)

		// PnC
		ocppGroup.POST("/get-installed-certificate-ids", ocpp.GetInstalledCertificateIds)
		ocppGroup.POST("/delete-certificate", ocpp.DeleteCertificateOnDevice)
		ocppGroup.POST("/install-certificate", ocpp.InstallCertificate)
		ocppGroup.POST("/trigger-message", ocpp.TriggerMessage)
		ocppGroup.POST("/sign-certificate", ocpp.SignCertificate)
		ocppGroup.POST("/certificate-signed", ocpp.CertificateSigned)
		ocppGroup.POST("/contract-cert-group", ocpp.SaveContractCertGroup)
	}

	// ── Users ─────────────────────────────────────────────────────────────────
	users := auth.Group("/users")
	{
		users.GET("", ListUsers)
		users.GET("/cpops", ListCPOPs) // CS_Admin: list all CP_OP
		users.GET("/cpoms", ListCPOMs) // CS_Admin/CP_OP: list CP_OM under given parent
		users.GET("/:id", GetUser)
		users.POST("", CreateUser)
		users.PUT("/:id", UpdateUser)
		users.PUT("/:id/permissions", middleware.RequireCSAdminOrCPOP(), UpdateUserPermissions)
		users.DELETE("/:id", DeleteUser) // role check inside handler
	}

	// ── Certificates library ─────────────────────────────────────────────────
	certs := auth.Group("/certificates")
	certs.Use(middleware.RequireCSAdminOrCPOP())
	{
		certs.GET("", ListCertificates)
		certs.POST("", UploadCertificate)
		certs.GET("/:id/content", GetCertificateContent)
		certs.PUT("/:id", RenameCertificate)
		certs.DELETE("/:id", DeleteCertificate)
	}

	// ── ID Tags ───────────────────────────────────────────────────────────────
	tags := auth.Group("/idtags")
	{
		tags.GET("", ListIDTags)
		tags.POST("", CreateIDTag)
		tags.PUT("/:id", UpdateIDTag)
		tags.DELETE("/:id", DeleteIDTag)
	}

	// ── Charging Profiles ────────────────────────────────────────────────────
	profiles := auth.Group("/profiles")
	{
		profiles.GET("", ListProfiles)
		profiles.POST("", UploadProfile)
		profiles.PUT("/:id", RenameProfile)
		profiles.DELETE("/:id", DeleteProfile)
	}

	// ── VDV261 ─────────────────────────────────────────────────────────────────
	vdv := auth.Group("/vdv")
	{
		vdv.GET("/profiles", ListVDVProfiles)
		vdv.POST("/profiles", CreateVDVProfile)
		vdv.PUT("/profiles/:id", UpdateVDVProfile)
		vdv.DELETE("/profiles/:id", DeleteVDVProfile)
		vdv.GET("/carinfos", ListVDVCarInfos)
		vdv.POST("/carinfos", CreateVDVCarInfo)
		vdv.PUT("/carinfos/:id", UpdateVDVCarInfo)
		vdv.DELETE("/carinfos/:id", DeleteVDVCarInfo)
		// Settings (CS_Admin only)
		vdv.GET("/settings", middleware.RequireCSAdmin(), GetVDVSettings)
		vdv.POST("/settings", middleware.RequireCSAdmin(), UpdateVDVSettings)
		vdv.POST("/settings/restart", middleware.RequireCSAdmin(), RestartVDVService)
		vdv.POST("/settings/upload-cert", middleware.RequireCSAdmin(), UploadVDVCert)
		vdv.GET("/settings/download-cert", middleware.RequireCSAdmin(), DownloadVDVCert)
	}

	// ── Health ───────────────────────────────────────────────────────────────
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
		})
	})

	return r
}
