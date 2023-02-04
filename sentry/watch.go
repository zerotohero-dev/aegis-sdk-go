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
	successThreshold, errorThreshold int64,
	interval, initialInterval, maxInterval time.Duration,
	factor int64,
) (time.Duration, int64, int64) {
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
			interval = time.Duration(int64(interval) / factor)
			// boundary check:
			if interval < initialInterval {
				interval = initialInterval
			}
		}

		return interval, successCount, errorCount
	}

	// Back off after N consecutive failures.
	if expandInterval {
		interval = time.Duration(int64(interval) * factor)
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
	maxInterval := env.SidecarMaxPollInterval()
	factor := env.SidecarExponentialBackoffMultiplier()
	successThreshold := env.SidecarSuccessThreshold()
	errorThreshold := env.SidecarErrorThreshold()

	interval := env.SidecarPollInterval()
	initialInterval := interval
	successCount := int64(0)
	errorCount := int64(0)
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
