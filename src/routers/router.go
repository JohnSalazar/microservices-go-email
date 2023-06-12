package routers

import (
	"github.com/JohnSalazar/microservices-go-common/config"
	"github.com/JohnSalazar/microservices-go-common/middlewares"
	common_service "github.com/JohnSalazar/microservices-go-common/services"
	"github.com/gin-contrib/location"
	"github.com/gin-gonic/gin"
)

type Router struct {
	config         *config.Config
	serviceMetrics common_service.Metrics
}

func NewRouter(
	config *config.Config,
	serviceMetrics common_service.Metrics,
) *Router {
	return &Router{
		config:         config,
		serviceMetrics: serviceMetrics,
	}
}

func (r *Router) RouterSetup() *gin.Engine {
	router := r.initRoute()
	router.Use(gin.Logger())
	router.Use(gin.Recovery())
	router.Use(middlewares.CORS())
	router.Use(location.Default())

	router.GET("/healthy", middlewares.Healthy())
	router.GET("/metrics", middlewares.MetricsHandler())

	return router
}

func (r *Router) initRoute() *gin.Engine {
	if r.config.Production {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	return gin.New()
}
