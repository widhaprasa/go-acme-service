package acme

import (
	"errors"
	"strings"
	"time"
)

func ValidateDomains(domains []string) ([]string, error) {

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

func GetTimeoutAndIntervalForDomain(domain string) (bool, time.Duration, time.Duration) {

	if strings.HasSuffix(domain,".id") {
		return false, 3600 * time.Second, 30  * time.Second
	}
	return true, 0, 0
}
