package a

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	certsrepository "github.com/widhaprasa/go-acme-service/repository/certs"
	webhookrepository "github.com/widhaprasa/go-acme-service/repository/webhook"
	certsservice "github.com/widhaprasa/go-acme-service/service/certs"
)

type CertsController struct {
	CertsRepository   certsrepository.CertsRepository
	CertsService      certsservice.CertsService
	WebhookRepository webhookrepository.WebhookRepository
}

func (c *CertsController) List(ctx *gin.Context) {

	list, err := c.CertsRepository.ListCerts()
	if err != nil {
		ctx.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	webhookMap, err := c.WebhookRepository.MapWebhook()
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

		certItem := map[string]any{
			"main":          main,
			"sans":          sans,
			"email":         email,
			"not_before_ts": notBeforeTs,
			"not_after_ts":  notAfterTs,
			"upserted_ts":   upsertedTs,
		}

		webhookItem, webhookOk := webhookMap[main].(map[string]any)
		if webhookOk {
			certItem["webhook_url"] = webhookItem["url"].(string)
			certItem["webhook_headers"] = webhookItem["headers"].(map[string]any)
		}

		certs = append(certs, certItem)
	}

	ctx.JSON(http.StatusOK, map[string]any{
		"certs": certs,
	})
}

func (c *CertsController) Read(ctx *gin.Context) {

	// Request body
	var data map[string]any
	if err := ctx.ShouldBindJSON(&data); err != nil {
		ctx.AbortWithStatus(http.StatusNotFound)
		return
	}

	domain, domainOk := data["domain"].(string)
	if !domainOk || domain == "" {
		ctx.AbortWithStatus(http.StatusBadRequest)
		return
	}

	// Retrieve from Db
	certs, err := c.CertsRepository.GetCerts(domain)
	if err != nil {
		ctx.AbortWithStatus(http.StatusNotFound)
		return
	}

	main := certs["main"].(string)
	sans := certs["sans"].(string)
	email := certs["email"].(string)
	notBeforeTs := certs["not_before_ts"].(int)
	notAfterTs := certs["not_after_ts"].(int)
	upsertedTs := certs["upserted_ts"].(int)

	certItem := map[string]any{
		"main":          main,
		"sans":          sans,
		"email":         email,
		"not_before_ts": notBeforeTs,
		"not_after_ts":  notAfterTs,
		"upserted_ts":   upsertedTs,
	}

	webhook, err := c.WebhookRepository.GetWebhook(main)
	if err == nil {
		certItem["webhook_url"] = webhook["url"].(string)
		certItem["webhook_headers"] = webhook["headers"].(map[string]any)
	}

	ctx.JSON(http.StatusOK, certItem)
}

func (c *CertsController) GetPrivateKey(ctx *gin.Context) {

	// Request body
	var data map[string]any
	if err := ctx.ShouldBindJSON(&data); err != nil {
		ctx.AbortWithStatus(http.StatusNotFound)
		return
	}

	domain, domainOk := data["domain"].(string)
	if !domainOk || domain == "" {
		ctx.AbortWithStatus(http.StatusBadRequest)
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

	domain, domainOk := data["domain"].(string)
	if !domainOk || domain == "" {
		ctx.AbortWithStatus(http.StatusBadRequest)
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

	webhookUrl, webhookUrlOk := data["webhook_url"].(string)
	if !webhookUrlOk {
		webhookUrl = ""
	}

	webhookHeaderMap, webhookHeaderMapOk := data["webhook_headers"].(map[string]any)
	if !webhookHeaderMapOk {
		webhookHeaderMap = map[string]any{}
	}

	// shouldCheckPropagation, scpOk := data["check_propagation"].(bool)
	// if !scpOk {
	// 	shouldCheckPropagation = true
	// }

	// Generate certs
	main, err := c.CertsService.GenerateCerts(ts, email, domains, webhookUrl, webhookHeaderMap)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, map[string]any{
			"message": err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, map[string]any{
		"main": main,
	})
}

func (c *CertsController) Delete(ctx *gin.Context) {

	// Server time
	// ts := time.Now().UnixMilli()

	// Request body
	var data map[string]any
	if err := ctx.ShouldBindJSON(&data); err != nil {
		ctx.AbortWithStatus(http.StatusNotFound)
		return
	}

	domain, domainOk := data["domain"].(string)
	if !domainOk || domain == "" {
		ctx.AbortWithStatus(http.StatusBadRequest)
		return
	}

	// Retrieve from Db
	certs, err := c.CertsRepository.GetCerts(domain)
	if err != nil {
		ctx.AbortWithStatus(http.StatusNotFound)
		return
	}
	main := certs["main"].(string)

	// Delete from Db
	_, err = c.CertsRepository.DeleteCerts(main)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, map[string]any{
			"message": err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, map[string]any{
		"main": main,
	})
}

func (c *CertsController) UpdateWebhook(ctx *gin.Context) {

	// Request body
	var data map[string]any
	if err := ctx.ShouldBindJSON(&data); err != nil {
		ctx.AbortWithStatus(http.StatusNotFound)
		return
	}

	domain, domainOk := data["domain"].(string)
	if !domainOk || domain == "" {
		ctx.AbortWithStatus(http.StatusBadRequest)
		return
	}

	// Retrieve from Db
	certs, err := c.CertsRepository.GetCerts(domain)
	if err != nil {
		ctx.AbortWithStatus(http.StatusNotFound)
		return
	}
	main := certs["main"].(string)

	url, urlOk := data["url"].(string)
	if !urlOk || url == "" {
		ctx.AbortWithStatus(http.StatusBadRequest)
		return
	}

	headerMap, headerMapOk := data["headers"].(map[string]any)
	if !headerMapOk {
		headerMap = map[string]any{}
	}

	_, err = c.WebhookRepository.UpsertWebhook(main, url, headerMap)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, map[string]any{
			"message": err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, map[string]any{
		"main": main,
	})
}

func (c *CertsController) DeleteWebhook(ctx *gin.Context) {

	// Request body
	var data map[string]any
	if err := ctx.ShouldBindJSON(&data); err != nil {
		ctx.AbortWithStatus(http.StatusNotFound)
		return
	}

	domain, domainOk := data["domain"].(string)
	if !domainOk || domain == "" {
		ctx.AbortWithStatus(http.StatusBadRequest)
		return
	}

	// Retrieve from Db
	certs, err := c.CertsRepository.GetCerts(domain)
	if err != nil {
		ctx.AbortWithStatus(http.StatusNotFound)
		return
	}
	main := certs["main"].(string)

	_, err = c.WebhookRepository.DeleteWebhook(main)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, map[string]any{
			"message": err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, map[string]any{
		"main": main,
	})
}
