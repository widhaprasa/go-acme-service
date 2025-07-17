package certs

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/go-acme/lego/v4/certificate"
	"github.com/widhaprasa/go-acme-service/repository/certs"
	"github.com/widhaprasa/go-acme-service/repository/webhook"
	"github.com/widhaprasa/go-acme-service/service/client"
)

type CertsService struct {
	certsRepository   certs.CertsRepository
	clientService     client.ClientService
	webhookRepository webhook.WebhookRepository
	jobs              chan map[string]any
}

func NewCertsService(certsrepository certs.CertsRepository, clientservice client.ClientService, webhookRepository webhook.WebhookRepository) CertsService {

	jobsNumber := 5 // Max job queues
	jobs := make(chan map[string]any, jobsNumber)

	return CertsService{
		certsRepository:   certsrepository,
		clientService:     clientservice,
		webhookRepository: webhookRepository,
		jobs:              jobs,
	}
}

func (c *CertsService) GenerateCerts(ts int64, email string, domains []string, webhookUrl string, webhookHeaderMap map[string]any) (string, error) {

	domains, err := c.validateDomains(domains)
	if err != nil {
		log.Println("No domain was given")
		return "", err
	}
	var main string

	certs, err := c.certsRepository.GetCertsByMain(domains)
	if err != nil {
		main = domains[0]
	} else {
		main = certs["main"].(string)
	}
	log.Println("Generate certs:", main)

	result := c.AddJob(map[string]any{
		"ts":              ts,
		"email":           email,
		"main":            main,
		"domains":         domains,
		"webhook_url":     webhookUrl,
		"webhook_headers": webhookHeaderMap,
	})

	if !result {
		return "", errors.New("Busy. Please try again later")
	}

	return main, nil
}

func (c *CertsService) generateCertsJob(ts int64, email string, main string, domains []string, webhookUrl string, webhookHeaderMap map[string]any) error {

	client, err := c.clientService.GetClient(ts, email, main)
	if err != nil {
		log.Println("Unable to get client:", email)
		return err
	}

	request := certificate.ObtainRequest{
		Domains:        domains,
		Bundle:         true,
		PreferredChain: "ISRG Root X1", // Default preferred chain
	}

	cert, err := client.Certificate.Obtain(request)
	if err != nil {
		log.Println("Error generating certificate for domain", main, ":", err)
		return err
	}
	if cert == nil {
		log.Println("Error generating certificate for domain", main, ":", err)
		return err
	}
	if len(cert.Certificate) == 0 || len(cert.PrivateKey) == 0 {
		log.Println("Certificate for domain", main, "is empty")
		return err
	}

	privateKey := cert.PrivateKey
	certificate_ := cert.Certificate

	res := certificate.Resource{
		Domain:      main,
		PrivateKey:  privateKey,
		Certificate: certificate_,
	}
	crt, _ := c.getX509Certificate(res)

	// Insert certs to database
	_, err = c.certsRepository.UpsertCerts(main, strings.Join(domains, ","), email, privateKey, certificate_,
		crt.NotBefore.UnixMilli(), crt.NotAfter.UnixMilli(), ts)
	if err != nil {
		log.Println("Failed to insert certs", main, ":", err)
		return err
	}

	// Push to webhook
	c.webhookPush("generate", main, email, privateKey, certificate_, webhookUrl, webhookHeaderMap)

	log.Println("Success generating certificate for domain", main)
	return nil
}

func (c *CertsService) RenewCerts(ts int64) error {

	log.Println("Run schedule renewing certificates...")

	renewPeriod := 30 * 24 * time.Hour // Default period, renew certificates if they are valid for less than a month

	list, err := c.certsRepository.ListCerts()
	if err != nil {
		log.Println("No domain was given")
		return err
	}

	for _, v := range list {

		certsMap := v.(map[string]any)
		main := certsMap["main"].(string)
		sans := certsMap["sans"].(string)
		privateKey := certsMap["private_key"].([]byte)
		certificate_ := certsMap["certificate"].([]byte)

		res := certificate.Resource{
			Domain:      main,
			PrivateKey:  privateKey,
			Certificate: certificate_,
		}

		crt, err := c.getX509Certificate(res)
		if err != nil || crt == nil || crt.NotAfter.Before(time.Now().Add(renewPeriod)) {

			// Renew certs
			log.Println("Renewing certificates:", main)

			email := certsMap["email"].(string)
			client, err := c.clientService.GetClient(ts, email, main)
			if err != nil {
				return err
			}

			opts := &certificate.RenewOptions{
				Bundle:         true,
				PreferredChain: "ISRG Root X1", // Default preferred chain
			}

			renewedCert, err := client.Certificate.RenewWithOptions(res, opts)
			if err != nil {
				log.Println("Error renewing certificate for domain", main, ":", err)
				return err
			}
			renewedCertificate := renewedCert.Certificate
			renewedPrivateKey := renewedCert.PrivateKey

			if len(renewedCertificate) == 0 || len(renewedPrivateKey) == 0 {
				log.Println("Certificate for domain", main, "is empty")
				return err
			}

			renewedRes := certificate.Resource{
				Domain:      main,
				PrivateKey:  renewedCertificate,
				Certificate: renewedPrivateKey,
			}
			renewedCrt, _ := c.getX509Certificate(renewedRes)

			// Update new certs to database
			_, err = c.certsRepository.UpsertCerts(main, sans, email, renewedCertificate, renewedPrivateKey,
				renewedCrt.NotBefore.UnixMilli(), renewedCrt.NotAfter.UnixMilli(), ts)
			if err != nil {
				log.Println("Failed to update certs", email, ":", err)
				return err
			}

			// Push to webhook
			c.webhookPush("renew", main, email, renewedCertificate, renewedPrivateKey, "", map[string]any{})

			log.Println("Success renewing certificate for domain", main)
		}
	}

	return nil
}

func (c *CertsService) getX509Certificate(res certificate.Resource) (*x509.Certificate, error) {

	tlsCert, err := tls.X509KeyPair(res.Certificate, res.PrivateKey)
	if err != nil {
		log.Println("Failed to load TLS key pair from ACME certificate for domain", res.Domain, ":", err, ". Certificate will be renewed")
		return nil, err
	}

	crt := tlsCert.Leaf
	if crt == nil {
		crt, err = x509.ParseCertificate(tlsCert.Certificate[0])
		if err != nil {
			log.Println("Failed to parse TLS key pair from ACME certificate for domain", res.Domain, ":", err, ". Certificate will be renewed")
		}
	}

	return crt, err
}

func (c *CertsService) validateDomains(domains []string) ([]string, error) {

	if len(domains) == 0 {
		return nil, errors.New("No domain was given")
	}

	map_ := make(map[string]struct{})

	var result []string
	for _, str := range domains {
		if _, exists := map_[str]; !exists {
			map_[str] = struct{}{}
			result = append(result, str)
		}
	}
	return result, nil
}

func (c *CertsService) webhookPush(type_ string, main string, email string, privateKey []byte, certificate_ []byte,
	webhookUrl string, webhookHeaderMap map[string]any) error {

	webhookBody, _ := json.Marshal(map[string]any{
		"type":        type_,
		"main":        main,
		"email":       email,
		"private_key": base64.StdEncoding.EncodeToString(privateKey),
		"certificate": base64.StdEncoding.EncodeToString(certificate_),
	})

	if webhookUrl == "" {

		// Retrieve webhook url from db
		webhook, err := c.webhookRepository.GetWebhook(main)
		if err != nil {
			return err
		}
		webhookUrl = webhook["url"].(string)
		webhookHeaderMap = webhook["headers"].(map[string]any)

	} else {

		// Update webhook url if specified
		_, err := c.webhookRepository.UpsertWebhook(main, webhookUrl, webhookHeaderMap)
		if err != nil {
			return err
		}
	}

	log.Println("Push webhook for domain:", main, "to:", webhookUrl, "type:", type_)

	// Push to webhook
	req, err := http.NewRequest("POST", webhookUrl, bytes.NewBuffer(webhookBody))
	if err != nil {
		return err
	}
	for key, value := range webhookHeaderMap {
		req.Header.Set(key, fmt.Sprintf("%v", value))
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return err
}
