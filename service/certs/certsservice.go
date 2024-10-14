package certs

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"log"
	"strings"
	"time"

	"github.com/go-acme/lego/v4/certificate"
	"github.com/go-acme/lego/v4/challenge/dns01"
	"github.com/go-acme/lego/v4/providers/dns/cloudflare"
	"github.com/widhaprasa/go-acme-service/repository/certs"
	"github.com/widhaprasa/go-acme-service/service/client"
)

type CertsService struct {
	CertsRepository *certs.CertsRepository
	ClientService   *client.ClientService
}

func (c *CertsService) GenerateCerts(ts int64, email string, domains []string, disablePropagationCheck bool) error {

	domains, err := c.validateDomains(domains)
	if err != nil {
		log.Println("No domain was given")
		return err
	}
	main := domains[0]

	client, err := c.ClientService.GetClient(ts, email)
	if err != nil {
		log.Println("Unable to get client:", email)
		return err
	}

	// Using cloudflare DNS provider
	dnsProvider, err := cloudflare.NewDNSProvider()
	if err != nil {
		log.Println("Unable to initiate Cloudflare DNS Provider:", err)
		return err
	}
	resolvers := []string{}

	err = client.Challenge.SetDNS01Provider(dnsProvider,
		dns01.CondOption(len(resolvers) > 0, dns01.AddRecursiveNameservers(resolvers)),
		dns01.WrapPreCheck(func(domain, fqdn, value string, check dns01.PreCheckFunc) (bool, error) {
			if disablePropagationCheck {
				return true, nil
			}
			return check(fqdn, value)
		}))
	if err != nil {
		log.Println("Unable to challenge use Cloudflare DNS Provider:", err)
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
		log.Printf("Certificate for domain %s is empty", main)
		return err
	}

	privateKey := cert.PrivateKey
	certificate_ := cert.Certificate

	res := certificate.Resource{
		Domain:      main,
		PrivateKey:  privateKey,
		Certificate: certificate_,
	}
	crt, err := c.getX509Certificate(res)

	// Insert certs to database
	_, err = c.CertsRepository.UpsertCerts(main, strings.Join(domains, ","), email, privateKey, certificate_, crt.NotBefore.UnixMilli(), crt.NotAfter.UnixMilli(), ts)
	if err != nil {
		log.Println("Failed to insert certs", email, ":", err)
		return err
	}

	return nil
}

func (c *CertsService) InitRenewTicker(ts int64) {

	renewInterval := 24 * time.Hour // Default interval, check renew certificates daily

	// Check renew first
	c.renewCerts(ts)

	ticker := time.NewTicker(renewInterval)
	done := make(chan bool)
	go func() {
		for {
			select {
			case <-ticker.C:
				ts := time.Now().UnixMilli()
				c.renewCerts(ts)
			case <-done:
				ticker.Stop()
				return
			}
		}
	}()
}

func (c *CertsService) renewCerts(ts int64) error {

	renewPeriod := 30 * 24 * time.Hour // Default period, renew certificates if they are valid for less than a month

	list, err := c.CertsRepository.ListCerts()
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
			log.Printf("Renew certs:", main)

			email := certsMap["email"].(string)
			client, err := c.ClientService.GetClient(ts, email)
			if err != nil {
				return err
			}

			opts := &certificate.RenewOptions{
				Bundle:         true,
				PreferredChain: "ISRG Root X1", // Default preferred chain
			}

			renewedCert, err := client.Certificate.RenewWithOptions(res, opts)
			if err != nil {
				log.Printf("Error renewing certificate for domain", main, ":", err)
				return err
			}

			if len(renewedCert.Certificate) == 0 || len(renewedCert.PrivateKey) == 0 {
				log.Printf("Certificate for domain %s is empty", main)
				return err
			}

			// Update new key to database
			_, err = c.CertsRepository.UpsertCerts(main, sans, email, renewedCert.PrivateKey, renewedCert.Certificate,
				crt.NotBefore.UnixMilli(), crt.NotAfter.UnixMilli(), ts)
			if err != nil {
				log.Println("Failed to update certs", email, ":", err)
				return err
			}
		}
	}

	return nil
}

func (c *CertsService) getX509Certificate(res certificate.Resource) (*x509.Certificate, error) {

	tlsCert, err := tls.X509KeyPair(res.Certificate, res.PrivateKey)
	if err != nil {
		log.Printf("Failed to load TLS key pair from ACME certificate for domain", res.Domain, ":", err, ". Certificate will be renewed")
		return nil, err
	}

	crt := tlsCert.Leaf
	if crt == nil {
		crt, err = x509.ParseCertificate(tlsCert.Certificate[0])
		if err != nil {
			log.Printf("Failed to parse TLS key pair from ACME certificate for domain", res.Domain, ":", err, ". Certificate will be renewed")
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
