package a

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	certsrepository "github.com/widhaprasa/go-acme-service/repository/certs"
	certsservice "github.com/widhaprasa/go-acme-service/service/certs"
)

type CertsController struct {
	CertsRepository certsrepository.CertsRepository
	CertsService    certsservice.CertsService
}

func (c *CertsController) List(ctx *gin.Context) {

	list, err := c.CertsRepository.ListCerts()
	if err != nil {
		ctx.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	certs := []any{}
	for _, v := range list {

		certsMap := v.(map[string]any)
		main := certsMap["main"].(string)
		sans := certsMap["sans"].(string)
		email := certsMap["email"].(string)
		notBeforeTs := certsMap["not_before_ts"].(int)
		notAfterTs := certsMap["not_after_ts"].(int)
		upsertedTs := certsMap["upserted_ts"].(int)

		certs = append(certs, map[string]any{
			"main":          main,
			"sans":          sans,
			"email":         email,
			"not_before_ts": notBeforeTs,
			"not_after_ts":  notAfterTs,
			"upserted_ts":   upsertedTs,
		})
	}

	ctx.JSON(http.StatusOK, map[string]any{
		"certs": certs,
	})
}

func (c *CertsController) GetPrivateKey(ctx *gin.Context) {

	// Request body
	var data map[string]any
	if err := ctx.ShouldBindJSON(&data); err != nil {
		ctx.AbortWithStatus(http.StatusNotFound)
		return
	}

	main, mainOk := data["main"].(string)
	if !mainOk || main == "" {
		ctx.AbortWithStatus(http.StatusNotFound)
		return
	}

	// Retrieve from Db
	certs, err := c.CertsRepository.GetCerts(main)
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

	main, mainOk := data["main"].(string)
	if !mainOk || main == "" {
		ctx.AbortWithStatus(http.StatusNotFound)
		return
	}

	// Retrieve from Db
	certs, err := c.CertsRepository.GetCerts(main)
	if err != nil {
		ctx.AbortWithStatus(http.StatusNotFound)
		return
	}

	ctx.Data(http.StatusOK, "text/plain", certs["certificate"].([]byte))
}

func (c *CertsController) Generate(ctx *gin.Context) {

	// Server time
	ts := time.Now().UnixMilli()

	// Request body
	var data map[string]any
	if err := ctx.ShouldBindJSON(&data); err != nil {
		ctx.AbortWithStatus(http.StatusNotFound)
		return
	}

	// Handle single domain and SANS
	var domains []string
	domain, domainOk := data["domain"].(string)
	if domainOk {
		domains = append(domains, domain)
	} else {
		domainsAny, domainsOk := data["domains"].([]any)
		if !domainsOk {
			ctx.AbortWithStatus(http.StatusNotFound)
			return
		}

		for _, v := range domainsAny {
			if str, ok := v.(string); ok {
				domains = append(domains, str)
			}
		}
	}

	email, emailOk := data["email"].(string)
	if !emailOk || email == "" {
		ctx.AbortWithStatus(http.StatusNotFound)
		return
	}

	// shouldCheckPropagation, scpOk := data["check_propagation"].(bool)
	// if !scpOk {
	// 	shouldCheckPropagation = true
	// }

	// Generate certs
	err := c.CertsService.GenerateCerts(ts, email, domains, false)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, map[string]any{
			"message": err.Error(),
		})
		return
	}
}
