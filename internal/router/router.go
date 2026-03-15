package router

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/silentpass/silentpass/internal/adapter/otp"
	"github.com/silentpass/silentpass/internal/adapter/telco"
	"github.com/silentpass/silentpass/internal/config"
	"github.com/silentpass/silentpass/internal/handler"
	"github.com/silentpass/silentpass/internal/middleware"
	"github.com/silentpass/silentpass/internal/repository"
	"github.com/silentpass/silentpass/internal/pkg/auth"
	"github.com/silentpass/silentpass/internal/service/policy"
	"github.com/silentpass/silentpass/internal/service/risk"
	"github.com/silentpass/silentpass/internal/service/verification"
	"github.com/silentpass/silentpass/internal/service/webhook"
)

// Deps holds external dependencies injected into the router.
type Deps struct {
	DB    *pgxpool.Pool  // nil = use in-memory repos
	Redis *redis.Client  // nil = use in-memory rate limiter
}

func New(cfg *config.Config, deps *Deps) *gin.Engine {
	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(gin.Logger())
	r.Use(gin.Recovery())
	r.Use(middleware.CORS())

	// --- Repositories ---
	var sessionRepo verification.SessionRepository
	var tenantResolver middleware.TenantResolver

	if deps != nil && deps.DB != nil {
		sessionRepo = repository.NewPGSessionRepo(deps.DB)
		tenantResolver = repository.NewPGTenantRepo(deps.DB)
	} else {
		memSessionRepo := repository.NewSessionRepo()
		memTenantRepo := repository.NewTenantRepo()
		sessionRepo = memSessionRepo
		tenantResolver = memTenantRepo
	}

	// --- Adapters ---
	telcoRouter := telco.NewRouter()
	telcoRouter.Register(telco.NewSandboxAdapter())

	otpRouter := otp.NewRouter()
	sandboxOTP := otp.NewSandboxProvider()
	otpRouter.Register("sms", sandboxOTP)
	otpRouter.Register("whatsapp", sandboxOTP)
	otpRouter.Register("voice", sandboxOTP)

	// --- JWT ---
	tokenSvc := auth.NewTokenService(cfg.JWTSecret, 5*time.Minute)

	// --- Services ---
	verificationSvc := verification.NewService(sessionRepo, telcoRouter, otpRouter)
	verificationSvc.SetTokenService(tokenSvc)
	policyEngine := policy.NewEngine()
	riskSvc := risk.NewService(telcoRouter, policyEngine)

	// --- Webhook ---
	webhookStore := webhook.NewMemoryStore()
	_ = webhook.NewService(webhookStore) // available for event emission

	// --- Handlers ---
	verificationHandler := handler.NewVerificationHandler(verificationSvc)
	riskHandler := handler.NewRiskHandler(riskSvc)
	webhookHandler := handler.NewWebhookHandler(webhookStore)
	policyHandler := handler.NewPolicyHandler()
	statsCollector := handler.NewStatsCollector()
	statsHandler := handler.NewStatsHandler(statsCollector)
	logsHandler := handler.NewLogsHandler(handler.NewLogStore())
	billingHandler := handler.NewBillingHandler()

	// --- Health ---
	r.GET("/health", func(c *gin.Context) {
		status := gin.H{"status": "ok", "service": "silentpass", "storage": "memory"}
		if deps != nil && deps.DB != nil {
			status["storage"] = "postgres"
			if err := deps.DB.Ping(c.Request.Context()); err != nil {
				status["status"] = "degraded"
				status["db_error"] = err.Error()
			}
		}
		c.JSON(200, status)
	})

	// --- API v1 ---
	v1 := r.Group("/v1")
	if deps != nil && deps.Redis != nil {
		v1.Use(middleware.RedisRateLimit(deps.Redis, 1000, time.Minute))
	} else {
		v1.Use(middleware.RateLimit(1000, time.Minute))
	}
	v1.Use(middleware.APIKeyAuth(tenantResolver))

	vg := v1.Group("/verification")
	{
		vg.POST("/session", verificationHandler.CreateSession)
		vg.POST("/silent", verificationHandler.SilentVerify)
		vg.POST("/otp/send", verificationHandler.SendOTP)
		vg.POST("/otp/check", verificationHandler.CheckOTP)
	}

	rg := v1.Group("/risk")
	{
		rg.POST("/sim-swap", riskHandler.SIMSwap)
		rg.POST("/verdict", riskHandler.Verdict)
	}

	// Webhook endpoints
	wg := v1.Group("/webhooks")
	{
		wg.POST("", webhookHandler.Create)
	}

	// Policy endpoints
	pg := v1.Group("/policies")
	{
		pg.GET("", policyHandler.List)
		pg.POST("", policyHandler.Create)
		pg.PUT("/:id", policyHandler.Update)
		pg.DELETE("/:id", policyHandler.Delete)
	}

	// Stats endpoints
	sg := v1.Group("/stats")
	{
		sg.GET("/dashboard", statsHandler.Dashboard)
		sg.GET("/activity", statsHandler.RecentActivity)
	}

	// Logs endpoint
	v1.GET("/logs", logsHandler.List)

	// Billing endpoint
	v1.GET("/billing/summary", billingHandler.Summary)

	return r
}
