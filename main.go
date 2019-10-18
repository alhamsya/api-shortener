package main

import (
	"api-shortener/cache"
	"api-shortener/config"
	"api-shortener/controller"
	"api-shortener/models"
	"api-shortener/utils"
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"log"
	"net/http"
	"os"
)

func init() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
}

func main() {
	router := gin.Default()
	os.Setenv("TZ", "Asia/Jakarta")
	config.DbInit()
	config.Db.AutoMigrate(&models.ShortUrlModel{}).AddUniqueIndex(
		"idx_create_at_email_user",
		"created_at", "email_user",
	)

	config.Db.AutoMigrate(&models.UserModel{}).AddUniqueIndex(
		"idx_create_at_email_user",
		"created_at", "email_user",
	)
	cache.RedisInit()

	router.GET("/", GetInfo)

	v1 := router.Group("/api/v1/")
	{
		v1.POST("/", controller.CreateShortUrl)
		v1.GET("/:param_short_url", controller.GetOneShortUrl)
	}

	oauth := router.Group("/oauth/")
	{
		oauth.GET("/login", controller.GoogleLogin)
		oauth.GET("/callback", controller.GoogleCallback)
		oauth.GET("/logout", controller.GoogleLogout)
	}

	router.Run(fmt.Sprintf("%s:%s",
		os.Getenv("URL_BACKEND"),
		os.Getenv("PORT_BACKEND"),
	))
}

func GetInfo(c *gin.Context) {
	q := c.Request.URL.Query()
	token := q.Get("token")

	if token == "" {
		c.JSON(http.StatusUnauthorized, utils.ErrMsg{
			Status:  false,
			Message: "Not authorization and access",
		})
		return
	}

	user, err := utils.GetSession(q.Get("token"))
	if user == nil {
		c.JSON(http.StatusUnauthorized, utils.ErrMsg{
			Status:  false,
			Message: err.Error(),
		})
		return
	}

	cache_jwt, _ := cache.GetValue("AUTH", user.Email)
	if cache_jwt == nil || cache_jwt != token {
		c.Redirect(http.StatusFound, "/")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token":        q.Get("token"),
		"id_g":         user.Id,
		"email":        user.Email,
		"ex_sess":      user.ExpiresAt,
		"current_time": utils.GetCurrentTime(),
	})
}

func AuthMiddleware(c *gin.Context) *controller.Claims {
	tokenString := c.Request.Header.Get("Authorization")
	claims := &controller.Claims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (i interface{}, e error) {
		if jwt.SigningMethodHS256 != token.Method {
			return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
		}
		jwt_key := []byte(os.Getenv("JWT_KEY"))
		return jwt_key, nil
	})

	sess_jwt, _ := cache.GetValue("AUTH", claims.Email)
	if sess_jwt != tokenString {
		return nil
	}

	// if token.Valid && err == nil {
	if token != nil && err == nil {
		return claims
	} else {
		return nil
	}
}
