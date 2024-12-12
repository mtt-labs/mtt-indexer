package router

import (
	"github.com/gin-gonic/gin"
	"mtt-indexer/controller"
	"mtt-indexer/service"
	"net/http"
)

func Init(s service.IService) *gin.Engine {
	r := gin.Default()
	group := r.Group("")

	group.Use(Cors())

	group.GET("/delegatorList", controller.DelegatorListEndpoint(s))
	group.GET("/delegatorHistory", controller.DelegatorHistoryEndpoint(s))
	group.GET("/validatorHistory", controller.ValidatorHistoryEndpoint(s))
	group.GET("/stakeHistory", controller.StakeHistoryEndpoint(s))
	group.GET("/commissionRecord", controller.CommissionRecordEndpoint(s))
	group.GET("/height", controller.HeightEndpoint(s))
	return r
}

func Cors() gin.HandlerFunc {
	return func(c *gin.Context) {
		method := c.Request.Method
		origin := c.Request.Header.Get("Origin")
		if origin != "" {
			c.Header("Access-Control-Allow-Origin", "*") // 可将将 * 替换为指定的域名
			c.Header("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE, UPDATE")
			c.Header("Access-Control-Allow-Headers", "Origin, X-Requested-With, Content-Type, Accept, Authorization")
			c.Header("Access-Control-Expose-Headers", "Content-Length, Access-Control-Allow-Origin, Access-Control-Allow-Headers, Cache-Control, Content-Language, Content-Type")
			c.Header("Access-Control-Allow-Credentials", "true")
		}
		if method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
		}
		c.Next()
	}
}
