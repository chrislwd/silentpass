package router

import (
	"log"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/silentpass/silentpass/internal/adapter/otp"
	"github.com/silentpass/silentpass/internal/adapter/telco"
	"github.com/silentpass/silentpass/internal/config"
	"github.com/silentpass/silentpass/internal/handler"
	"github.com/silentpass/silentpass/internal/middleware"
	"github.com/silentpass/silentpass/internal/pkg/auth"
	"github.com/silentpass/silentpass/internal/repository"
	"github.com/silentpass/silentpass/internal/service/policy"
	"github.com/silentpass/silentpass/internal/service/risk"
	"github.com/silentpass/silentpass/internal/service/verification"
	"github.com/silentpass/silentpass/internal/service/webhook"
)

// Deps holds external dependencies injected into the router.
type Deps struct {
	DB    *pgxpool.Pool // nil = use in-memory repos
	Redis *redis.Client // nil = use in-memory rate limiter
}

func New(cfg *config.Config, deps *Deps) *gin.Engine {
	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	hasPG := deps != nil && deps.DB != nil

	r := gin.New()
	r.Use(gin.Logger())
	r.Use(gin.Recovery())
	r.Use(middleware.CORS())

	// --- Repositories ---
	var sessionRepo verification.SessionRepository
	var tenantResolver middleware.TenantResolver

	if hasPG {
		sessionRepo = repository.NewPGSessionRepo(deps.DB)
		tenantResolver = repository.NewPGTenantRepo(deps.DB)
	} else {
		sessionRepo = repository.NewSessionRepo()
		tenantResolver = repository.NewTenantRepo()
	}

	// --- Store Interfaces (PG or in-memory) ---
	var policyStore handler.PolicyStore
	var logsStore handler.LogsStore
	var billingStore handler.BillingStore
	var statsStore handler.StatsStore

	if hasPG {
		policyStore = repository.NewPGPolicyRepo(deps.DB)
		logsStore = handler.NewPGLogsStore(deps.DB)
		billingStore = handler.NewPGBillingStore(deps.DB)
		statsStore = handler.NewPGStatsStore(deps.DB)
		log.Println("using PostgreSQL for all stores")
	} else {
		policyStore = handler.NewMemoryPolicyStore()
		logsStore = handler.NewMemoryLogsStore()
		billingStore = handler.NewMemoryBillingStore()
		statsStore = nil // will use in-memory collector fallback
		log.Println("using in-memory stores")
	}

	// --- Telco Adapters ---
	telcoRouter := telco.NewRouter()
	if cfg.ProvidersConfig != "" {
		providersCfg, err := telco.LoadProvidersConfig(cfg.ProvidersConfig)
		if err != nil {
			log.Printf("WARNING: failed to load providers config: %v, using sandbox", err)
			telcoRouter.Register(telco.NewSandboxAdapter())
		} else {
			if err := telco.BuildAdapters(providersCfg, telcoRouter); err != nil {
				log.Printf("WARNING: failed to build adapters: %v, using sandbox", err)
				telcoRouter.Register(telco.NewSandboxAdapter())
			} else {
				log.Printf("loaded %d telco provider(s) from config", len(providersCfg.Providers))
			}
		}
	} else {
		telcoRouter.Register(telco.NewSandboxAdapter())
	}

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
	_ = webhook.NewService(webhookStore)

	// --- Handlers ---
	verificationHandler := handler.NewVerificationHandler(verificationSvc)
	riskHandler := handler.NewRiskHandler(riskSvc)
	webhookHandler := handler.NewWebhookHandler(webhookStore)
	policyHandler := handler.NewPolicyHandler(policyStore)
	statsCollector := handler.NewStatsCollector()
	statsHandler := handler.NewStatsHandler(statsStore, statsCollector)
	logsHandler := handler.NewLogsHandler(logsStore)
	billingHandler := handler.NewBillingHandler(billingStore)

	// --- Health ---
	r.GET("/health", func(c *gin.Context) {
		status := gin.H{"status": "ok", "service": "silentpass", "storage": "memory"}
		if hasPG {
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

	v1.POST("/webhooks", webhookHandler.Create)

	pg := v1.Group("/policies")
	{
		pg.GET("", policyHandler.List)
		pg.POST("", policyHandler.Create)
		pg.PUT("/:id", policyHandler.Update)
		pg.DELETE("/:id", policyHandler.Delete)
	}

	sg := v1.Group("/stats")
	{
		sg.GET("/dashboard", statsHandler.Dashboard)
		sg.GET("/activity", statsHandler.RecentActivity)
	}

	v1.GET("/logs", logsHandler.List)
	v1.GET("/billing/summary", billingHandler.Summary)

	return r
}
