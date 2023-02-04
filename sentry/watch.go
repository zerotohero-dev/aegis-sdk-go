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

var maxInterval = env.SidecarMaxPollInterval()
var factor = env.SidecarExponentialBackoffMultiplier()
var successThreshold = env.SidecarSuccessThreshold()
var errorThreshold = env.SidecarErrorThreshold()
var initialInterval = env.SidecarPollInterval()

func exponentialBackoff(
	success bool, interval time.Duration, successCount, errorCount int64,
) (time.Duration, int64, int64) {
	// #region Boundary Corrections
	if factor < 1 {
		factor = 1
	}
	if initialInterval > maxInterval {
		initialInterval = maxInterval
	}
	// #endregion

	// Decide whether to shrink, expand, or keep the interval the same
	// based on the success and error count so far.
	if success {
		nextSuccessCount := successCount + 1

		// We have a success, so the interval “may” shrink.
		shrinkInterval := nextSuccessCount >= successThreshold
		if shrinkInterval {
			nextInterval := time.Duration(int64(interval) / factor)

			// boundary check:
			if nextInterval < initialInterval {
				nextInterval = initialInterval
			}

			// Interval shrank.
			return nextInterval, 0, 0
		}

		// Success count increased, interval is intact.
		return interval, nextSuccessCount, 0
	}

	nextErrorCount := errorCount + 1

	// We have an error, so the interval “may” expand.
	expandInterval := nextErrorCount >= errorThreshold
	if expandInterval {
		nextInterval := time.Duration(int64(interval) * factor)

		// boundary check:
		if nextInterval > maxInterval {
			nextInterval = maxInterval
		}

		// Interval expanded.
		return nextInterval, 0, 0
	}

	// Error count increased, interval is intact.
	return interval, 0, nextErrorCount
}

// Watch synchronizes the internal state of the sidecar by talking to
// Aegis Safe regularly. It periodically calls Fetch behind-the-scenes to
// get its work done. Once it fetches the secrets, it saves it to
// the location defined in the `AEGIS_SIDECAR_SECRETS_PATH` environment
// variable (`/opt/aegis/secrets.json` by default).
func Watch() {
	interval := initialInterval
	successCount := int64(0)
	errorCount := int64(0)

	for {
		ticker := time.NewTicker(interval)
		select {
		case <-ticker.C:
			err := fetchSecrets()

			// Update parameters based on success/failure.
			interval, successCount, errorCount = exponentialBackoff(
				err == nil, interval, successCount, errorCount,
			)

			if err != nil {
				log.InfoLn("Could not fetch secrets", err.Error(),
					". Will retry in", interval, ".")
			}

			ticker.Stop()
		}
	}
}
