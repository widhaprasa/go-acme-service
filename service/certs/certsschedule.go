package certs

import "time"

func (c *CertsService) InitRenewSchedule(ts int64) {

	renewInterval := 24 * time.Hour // Default interval, check renew certificates daily

	// Check renew first
	c.RenewCerts(ts)

	ticker := time.NewTicker(renewInterval)
	done := make(chan bool)
	go func() {
		for {
			select {
			case <-ticker.C:
				ts := time.Now().UnixMilli()
				c.RenewCerts(ts)
			case <-done:
				ticker.Stop()
				return
			}
		}
	}()
}

func (c *CertsService) InitJobSchedule() {

	go func() {
		for payload := range c.jobs {

			ts := payload["ts"].(int64)
			email := payload["email"].(string)
			main := payload["main"].(string)
			domains := payload["domains"].([]string)
			webhookUrl := payload["webhook_url"].(string)
			webhookHeaderMap := payload["webhook_headers"].(map[string]any)

			err := c.generateCertsJob(ts, email, main, domains, webhookUrl, webhookHeaderMap)
			if err != nil {

			}
		}
	}()
}

func (c *CertsService) AddJob(payload map[string]any) bool {

	select {
	case c.jobs <- payload:
		return true
	default:
		return false
	}
}
