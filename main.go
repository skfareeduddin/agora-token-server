package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	rtctokenbuilder2 "github.com/AgoraIO-Community/go-tokenbuilder/rtctokenbuilder"
	rtmtokenbuilder2 "github.com/AgoraIO-Community/go-tokenbuilder/rtmtokenbuilder"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

var appID, appCertificate string

func init() {
	os.Setenv("APP_ID", "481d40e954fd474ab5157dad831108a4")
	os.Setenv("APP_CERTIFICATE", "0df91a56f4404004996efef86e288b84")

	if err := godotenv.Load(); err != nil {
		log.Print("No .env file found")
	}
}

func main() {

	appIDEnv, appIDExists := os.LookupEnv("APP_ID")
	appCertificateEnv, appCertificateExists := os.LookupEnv("APP_CERTIFICATE")

	if !appIDExists || !appCertificateExists {
		log.Fatal("FATAL ERROR: ENV not properly configured, check APP_ID and APP_CERTIFICATE")
	} else {
		appID = appIDEnv
		appCertificate = appCertificateEnv
	}

	api := gin.Default()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	api.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "pong",
		})
	})

	api.Use(nocache())
	api.GET("rtc/:channelName/:role/:tokenType/:uid/", getRtcToken)
	api.GET("rtm/:uid/", getRtmToken)
	api.GET("rte/:channelName/:role/:tokenType/:uid/", getBothTokens)

	api.Run(":" + port)
}

func nocache() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Cache-Control", "private, no-cache, no-store, must-revalidate")
		c.Header("Expires", "-1")
		c.Header("Pragma", "no-cache")
		c.Header("Access-Control-Allow-Origin", "*")
	}
}

func getRtcToken(c *gin.Context) {
	channelName, tokenType, uidStr, role, expireTimestamp, err := parseRtcParams(c)

	if err != nil {
		c.Error(err)
		c.AbortWithStatusJSON(400, gin.H{
			"message": "Error generating RTC token:" + err.Error(),
			"status":  400,
		})
		return
	}

	rtcToken, tokenErr := generateRtcToken(channelName, uidStr, tokenType, role, expireTimestamp)

	if tokenErr != nil {
		log.Println(tokenErr)
		c.Error(err)
		c.AbortWithStatusJSON(400, gin.H{
			"status": 400,
			"error":  "Error generating RTC token: " + tokenErr.Error(),
		})
	} else {
		c.JSON(200, gin.H{
			"rtcToken": rtcToken,
		})
	}
}

func getRtmToken(c *gin.Context) {
	uidStr, expireTimestamp, err := parseRtmParams(c)

	if err != nil {
		c.Error(err)
		c.AbortWithStatusJSON(400, gin.H{
			"status":  400,
			"message": "Error generating RTM token" + err.Error(),
		})
		return
	}

	rtmToken, tokenErr := rtmtokenbuilder2.BuildToken(appID, appCertificate, uidStr, expireTimestamp, "")

	if tokenErr != nil {
		log.Println(err)
		c.Error(err)
		errMsg := "Error generating RTM token: " + tokenErr.Error()
		c.AbortWithStatusJSON(400, gin.H{
			"status": 400,
			"error":  errMsg,
		})
	} else {
		c.JSON(200, gin.H{
			"rtmToken": rtmToken,
		})
	}
}

func getBothTokens(c *gin.Context) {
	channelName, tokenType, uidStr, role, expireTimestamp, rtcParamErr := parseRtcParams(c)

	if rtcParamErr != nil {
		c.Error(rtcParamErr)
		c.AbortWithStatusJSON(400, gin.H{
			"status":  400,
			"message": "Error generating tokens: " + rtcParamErr.Error(),
		})
	}

	rtcToken, rtcTokenErr := generateRtcToken(channelName, uidStr, tokenType, role, expireTimestamp)

	rtmToken, rtmTokenErr := rtmtokenbuilder2.BuildToken(appID, appCertificate, uidStr, expireTimestamp, "")

	if rtcTokenErr != nil {
		c.Error(rtcTokenErr)
		errMsg := "Error generating RTC token: " + rtcTokenErr.Error()
		c.AbortWithStatusJSON(400, gin.H{
			"status":  400,
			"message": errMsg,
		})
	} else if rtmTokenErr != nil {
		c.Error(rtmTokenErr)
		errMsg := "Error generating RTM token: " + rtmTokenErr.Error()
		c.AbortWithStatusJSON(400, gin.H{
			"status":  400,
			"message": errMsg,
		})
	} else {
		c.JSON(200, gin.H{
			"rtcToken": rtcToken,
			"rtmToken": rtmToken,
		})
	}
}

func parseRtcParams(c *gin.Context) (channelName, tokenType, uidStr string, role rtctokenbuilder2.Role, expireTimestamp uint32, err error) {
	channelName = c.Param("channelName")
	roleStr := c.Param("role")
	tokenType = c.Param("tokenType")
	uidStr = c.Param("uid")
	expireTime := c.DefaultQuery("expiry", "3600")

	if roleStr == "publisher" {
		role = rtctokenbuilder2.RolePublisher
	} else {
		role = rtctokenbuilder2.RoleSubscriber
	}

	expireTime64, parseErr := strconv.ParseUint(expireTime, 10, 64)

	if parseErr != nil {
		err = fmt.Errorf("failed to parse expireTime: %s, causing error:m %s", expireTime, parseErr)
	}

	expireTimeInSeconds := uint32(expireTime64)
	currentTimestamp := uint32(time.Now().UTC().Unix())
	expireTimestamp = currentTimestamp + expireTimeInSeconds

	return channelName, tokenType, uidStr, role, expireTimestamp, err
}

func parseRtmParams(c *gin.Context) (uidStr string, expireTimestamp uint32, err error) {
	uidStr = c.Param("uid")
	expireTime := c.DefaultQuery("expiry", "3600")

	expireTime64, parseErr := strconv.ParseUint(expireTime, 10, 64)

	if parseErr != nil {
		err = fmt.Errorf("failed to parse expireTime: %s, causing error:m %s", expireTime, parseErr)
	}

	expireTimeInSeconds := uint32(expireTime64)
	currentTimestamp := uint32(time.Now().UTC().Unix())
	expireTimestamp = currentTimestamp + expireTimeInSeconds

	return uidStr, expireTimestamp, err
}

func generateRtcToken(channelName, uidStr, tokenType string, role rtctokenbuilder2.Role, expireTimestamp uint32) (rtcToken string, err error) {
	log.Printf(appID, appCertificate)
	if tokenType == "userAccount" {
		rtcToken, err = rtctokenbuilder2.BuildTokenWithAccount(appID, appCertificate, channelName, uidStr, role, expireTimestamp)
		return rtcToken, err
	} else if tokenType == "uid" {
		uid64, parseErr := strconv.ParseUint(uidStr, 10, 64)

		if parseErr != nil {
			err = fmt.Errorf("failed to parse uidStr: %s, to unit causing error: %s", uidStr, parseErr)
			return "", err
		}

		uid := uint32(uid64)
		rtcToken, err = rtctokenbuilder2.BuildTokenWithUid(appID, appCertificate, channelName, uid, role, expireTimestamp)
		return rtcToken, err
	} else {
		err = fmt.Errorf("failed to generate RTC token for unknown tokenType: %s", tokenType)
		log.Println(err)
		return "", err
	}
}
