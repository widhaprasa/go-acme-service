# Let's Encrypt Certificate Generation and Renewal Service

`go-acme-service` is a service for generating and renewing Let's Encrypt certificates as a service. It utilizes the [`go-acme/lego`](https://github.com/go-acme/lego) library and currently supports the DNS Challenge using the Cloudflare provider.

The service is protected using basic authentication for its endpoints.

## How to Use

To use `go-acme-service`, you must set the following environment variables:

### Cloudflare API Credentials:
- `CF_API_EMAIL`
- `CF_DNS_API_TOKEN`
- `CF_ZONE_API_TOKEN`

These credentials are required to authenticate with Cloudflare's DNS for the DNS Challenge.

### Basic Authentication Credentials:
- `SERVICE_USERNAME`
- `SERVICE_PASSWORD`

These variables are used to secure the service endpoints with basic authentication.

### Service Port
The default port for the service is **8080**, but you can change it using the environment variable:
- `SERVICE_PORT`

## API Endpoints

| Description                         | Method | Endpoint                |
|-------------------------------------|--------|-------------------------|
| Certs List                           | GET    | `/certs/list`           |
| Certs Private Key                    | POST   | `/certs/privatekey`     |
| Certs Certificates                   | POST   | `/certs/certificate`    |
| Certs Generate                       | POST   | `/certs/generate`       |

For more details on how to configure the Cloudflare provider, please refer to the official documentation:  
[Cloudflare DNS Challenge Setup](https://go-acme.github.io/lego/dns/cloudflare/)