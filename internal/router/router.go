package router

// LEARNING NOTE: Bu dosya uygulamanın URL haritasıdır.
// Request önce buraya gelir, sonra HTTP method + path'e göre ilgili handler fonksiyonuna yönlendirilir.

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/nazazx/league-simulation/internal/handler"
)

// Setup creates the Gin router with all API routes.
func Setup(h *handler.Handler) *gin.Engine {
	// LEARNING NOTE: Gin HTTP framework'tür; gelen request'leri tanımlı route'lara göre handler'a yönlendirir.
	// *gin.Engine, Gin'in ana router/server nesnesidir.
	r := gin.Default()

	// CORS middleware
	// LEARNING NOTE: Middleware, request handler'a gitmeden önce çalışan ara katmandır; burada CORS header'ları eklenir.
	// `func(c *gin.Context)` anonim fonksiyondur; Gin her request için bu fonksiyonu çalıştırır.
	r.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type")
		if c.Request.Method == "OPTIONS" {
			// LEARNING NOTE: OPTIONS request browser'ın CORS preflight kontrolüdür; gerçek endpoint'e gitmeden 204 döner.
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	})

	// Serve frontend static files
	// LEARNING NOTE: Go backend sadece API değil, frontend dosyalarını da aynı porttan servis ediyor.
	r.StaticFile("/", "./frontend/index.html")
	r.Static("/static", "./frontend/static")

	// API v1 group
	// LEARNING NOTE: /api/v1 versiyonlama prefix'idir; ileride v2 çıkarsa eski endpoint'ler korunabilir.
	v1 := r.Group("/api/v1")
	{
		// LEARNING NOTE: GET veri okuma, POST yeni işlem başlatma, PUT güncelleme için kullanılır.
		// Her satır bir endpoint'i handler metoduna bağlar.
		v1.GET("/health", h.HealthCheck)
		v1.GET("/teams", h.GetTeams)
		v1.POST("/teams", h.SaveTeam)
		v1.POST("/teams/import", h.ImportTeams)
		v1.DELETE("/teams/:id", h.DeleteTeam)
		v1.GET("/matches", h.GetMatches)
		v1.GET("/matches/week/:week", h.GetMatchesByWeek)
		v1.GET("/historical-matches", h.GetHistoricalMatches)
		v1.GET("/historical-stats", h.GetHistoricalStats)
		v1.PUT("/matches/:id", h.UpdateMatchResult)
		v1.POST("/fixtures/generate", h.GenerateFixtures)
		v1.POST("/play/week", h.PlayNextWeek)
		v1.POST("/play/all", h.PlayAllRemaining)
		v1.GET("/standings", h.GetStandings)
		v1.GET("/predictions", h.GetPredictions)
		v1.POST("/agent/chat", h.AgentChat)
		v1.POST("/reset", h.Reset)
	}

	return r
}
