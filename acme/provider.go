package acme

import (
	"fmt"
	"time"

	"github.com/go-acme/lego/v4/providers/dns/cloudflare"
)

type CloudflareDNSCustomTimeoutProvider struct {
	*cloudflare.DNSProvider
	timeout  time.Duration
	interval time.Duration
}

func NewCloudflareDNSCustomTimeoutProvider(
	timeout time.Duration,
	interval time.Duration,
) (*CloudflareDNSCustomTimeoutProvider, error) {
	provider, err := cloudflare.NewDNSProvider()
	if err != nil {
		return nil, fmt.Errorf("cloudflare: failed to create Cloudflare DNS provider: %w", err)
	}

	return &CloudflareDNSCustomTimeoutProvider{
		DNSProvider: provider,
		timeout:     timeout,
		interval:    interval,
	}, nil
}

func (p *CloudflareDNSCustomTimeoutProvider) Timeout() (timeout, interval time.Duration) {
	fmt.Printf("Using custom timeout: %s, interval: %s\n", p.timeout, p.interval)
	return p.timeout, p.interval
}
