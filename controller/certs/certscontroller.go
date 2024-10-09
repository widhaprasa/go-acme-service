package a

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	certsrepository "github.com/widhaprasa/go-acme-service/repository/certs"
	certsservice "github.com/widhaprasa/go-acme-service/service/certs"
)

type CertsController struct {
	CertsRepository *certsrepository.CertsRepository
	CertsService    *certsservice.CertsService
}

func (c *CertsController) List(ctx *gin.Context) {

	list, err := c.CertsRepository.ListCerts()
	if err != nil {
		ctx.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	result := []any{}
	for _, v := range list {

		certsMap := v.(map[string]any)
		domain := certsMap["domain"].(string)
		email := certsMap["email"].(string)
		notBeforeTs := certsMap["not_before_ts"].(int)
		notAfterTs := certsMap["not_after_ts"].(int)
		upsertedTs := certsMap["upserted_ts"].(int)

		result = append(result, map[string]any{
			"domain":        domain,
			"email":         email,
			"not_before_ts": notBeforeTs,
			"not_after_ts":  notAfterTs,
			"upserted_ts":   upsertedTs,
		})
	}

	ctx.JSON(http.StatusOK, result)
}

func (c *CertsController) GetPrivateKey(ctx *gin.Context) {

	// Request body
	var data map[string]any
	if err := ctx.ShouldBindJSON(&data); err != nil {
		ctx.AbortWithStatus(http.StatusNotFound)
		return
	}

	domain := data["domain"].(string)
	if domain == "" {
		ctx.AbortWithStatus(http.StatusNotFound)
		return
	}

	// Retrieve from Db
	certs, err := c.CertsRepository.GetCerts(domain)
	if err != nil {
		ctx.AbortWithStatus(http.StatusNotFound)
		return
	}

	ctx.Data(http.StatusOK, "text/plain", certs["private_key"].([]byte))
}

func (c *CertsController) GetCertificate(ctx *gin.Context) {

	// Request body
	var data map[string]any
	if err := ctx.ShouldBindJSON(&data); err != nil {
		ctx.AbortWithStatus(http.StatusNotFound)
		return
	}

	domain := data["domain"].(string)
	if domain == "" {
		ctx.AbortWithStatus(http.StatusNotFound)
		return
	}

	// Retrieve from Db
	certs, err := c.CertsRepository.GetCerts(domain)
	if err != nil {
		ctx.AbortWithStatus(http.StatusNotFound)
		return
	}

	ctx.Data(http.StatusOK, "text/plain", certs["certificate"].([]byte))
}

func (c *CertsController) Generate(ctx *gin.Context) {

	// Server time
	ts := time.Now().UnixNano() / 1e6

	// Request body
	var data map[string]any
	if err := ctx.ShouldBindJSON(&data); err != nil {
		ctx.AbortWithStatus(http.StatusNotFound)
		return
	}

	domain := data["domain"].(string)
	email := data["email"].(string)
	if domain == "" || email == "" {
		ctx.AbortWithStatus(http.StatusNotFound)
		return
	}

	// Generate certs
	err := c.CertsService.GenerateCerts(ts, email, domain, false)
	if err != nil {
		ctx.AbortWithStatus(http.StatusInternalServerError)
		return
	}
}
