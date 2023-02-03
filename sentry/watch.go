/*
 * .-'_.---._'-.
 * ||####|(__)||   Protect your secrets, protect your business.
 *   \\()|##//       Secure your sensitive data with Aegis.
 *    \\ |#//                  <aegis.z2h.dev>
 *     .\_/.
 */

package sentry

import (
	"github.com/zerotohero-dev/aegis-core/env"
	"github.com/zerotohero-dev/aegis-core/log"
	"time"
)

func exponentialBackoff(err error, successCount, errorCount,
	successThreshold, errorThreshold int,
	interval, initialInterval, maxInterval time.Duration,
	factor int,
) (time.Duration, int, int) {
	shrinkInterval := false
	expandInterval := false
	if err == nil {
		successCount++
		errorCount = 0
		if successCount >= successThreshold {
			shrinkInterval = true
			successCount = 0
		}
	} else {
		errorCount++
		successCount = 0
		if errorCount >= errorThreshold {
			expandInterval = true
			errorCount = 0
		}
	}

	if err == nil {
		// Reduce interval after N consecutive successes.
		if shrinkInterval {
			interval = time.Duration(int(interval) / factor)
			// boundary check:
			if interval < initialInterval {
				interval = initialInterval
			}
		}

		return interval, successCount, errorCount
	}

	// Back off after N consecutive failures.
	if expandInterval {
		interval = time.Duration(int(interval) * factor)
		// boundary check:
		if interval > maxInterval {
			interval = maxInterval
		}
	}

	return interval, successCount, errorCount
}

// Watch synchronizes the internal state of the sidecar by talking to
// Aegis Safe regularly. It periodically calls Fetch behind-the-scenes to
// get its work done. Once it fetches the secrets, it saves it to
// the location defined in the `AEGIS_SIDECAR_SECRETS_PATH` environment
// variable (`/opt/aegis/secrets.json` by default).
func Watch() {
	maxInterval := env.SentinelMaxPollInterval()         // time.Minute * 2
	factor := env.SentinelExponentialBackOffMultiplier() // 2, 1.5
	successThreshold := env.SentinelSuccessThreshold()   // 3
	errorThreshold := env.SentinelErrorThreshold()       // 2

	interval := env.SentryPollInterval() // TODO: this should be milliseconds; right now it is seconds:: BREAKING CHANGE
	initialInterval := interval
	successCount := 0
	errorCount := 0
	for {
		ticker := time.NewTicker(interval)
		select {
		case <-ticker.C:
			err := fetchSecrets()

			// Update parameters based on success/failure.
			interval, successCount, errorCount = exponentialBackoff(
				err, successCount, errorCount, successThreshold,
				errorThreshold, interval, initialInterval, maxInterval, factor,
			)

			if err != nil {
				log.InfoLn("Could not fetch secrets", err.Error(),
					". Will retry in", interval, ".")
			}

			ticker.Stop()
		}
	}
}
