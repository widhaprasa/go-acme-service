package client

import (
	"fmt"
	"log"

	"github.com/go-acme/lego/v4/certcrypto"
	"github.com/go-acme/lego/v4/challenge/dns01"
	"github.com/go-acme/lego/v4/lego"
	"github.com/go-acme/lego/v4/providers/dns/cloudflare"
	"github.com/go-acme/lego/v4/registration"
	"github.com/widhaprasa/go-acme-service/acme"
	"github.com/widhaprasa/go-acme-service/repository/client"
)

type ClientService struct {
	Clientrepository client.ClientRepository
}

func (c *ClientService) GetClient(ts int64, email string) (*lego.Client, error) {

	// Using production CA server
	caServer := lego.LEDirectoryProduction
	var client *lego.Client

	clientMap, err := c.Clientrepository.GetClient(email)
	if err != nil {
		log.Println("Create new user:", email)

		user, err := acme.NewUser(email)
		if err != nil {
			log.Println("Unable to create user", email, ":", err)
			return nil, err
		}

		// Config for request to LE server
		config := lego.NewConfig(user)
		config.CADirURL = caServer
		config.Certificate.KeyType = certcrypto.RSA4096
		config.UserAgent = fmt.Sprintf("widhaprasa-acme/%s", "1.0")

		// Create ACME client
		client, err = lego.NewClient(config)
		if err != nil {
			log.Println("Unable to create ACME client", email, ":", err)
			return nil, err
		}

		// Register ACME client first
		res, err := client.Registration.Register(registration.RegisterOptions{TermsOfServiceAgreed: true})
		if err != nil {
			log.Println("Unable to register ACME client", email, ":", err)
			return nil, err
		}
		user.Registration = res

		// Save client to database
		_, err = c.Clientrepository.UpsertClient(email, user.Registration.URI, user.PrivateKey, ts)
		if err != nil {
			log.Println("Failed to insert client", email, ":", err)
			return nil, err
		}

	} else {

		email = clientMap["email"].(string)
		uri := clientMap["uri"].(string)
		privateKey := clientMap["private_key"].([]byte)

		user, err := acme.NewUserFull(email, uri, privateKey)
		if err != nil {
			log.Println("Unable to create user", email, ":", err)
			return nil, err
		}

		// Config for request to LE server
		config := lego.NewConfig(user)
		config.CADirURL = caServer
		config.Certificate.KeyType = certcrypto.RSA4096
		config.UserAgent = fmt.Sprintf("widhaprasa-acme/%s", "1.0")

		// Create ACME client
		client, err = lego.NewClient(config)
		if err != nil {
			log.Println("Unable to create ACME client", email, ":", err)
			return nil, err
		}

		// Register ACME client first
		res, err := client.Registration.QueryRegistration()
		if err != nil {
			log.Println("Unable to register ACME client", email, ":", err)
			return nil, err
		}
		user.Registration = res
	}

	// Using cloudflare DNS provider
	dnsProvider, err := cloudflare.NewDNSProvider()
	if err != nil {
		log.Println("Unable to initiate Cloudflare DNS Provider:", err)
		return nil, err
	}
	resolvers := []string{}

	// Set DNS-01 challenge timeout and retry intervals
	dns01.SetResolverWaitTime(600 * time.Second)
	dns01.SetResolverPreCheckTimeout(10 * time.Second) 

	err = client.Challenge.SetDNS01Provider(dnsProvider,
		dns01.CondOption(len(resolvers) > 0, dns01.AddRecursiveNameservers(resolvers)),
		dns01.WrapPreCheck(func(domain, fqdn, value string, check dns01.PreCheckFunc) (bool, error) {
			return check(fqdn, value)
		}))
	if err != nil {
		log.Println("Unable to challenge use Cloudflare DNS Provider:", err)
		return nil, err
	}

	return client, nil
}
