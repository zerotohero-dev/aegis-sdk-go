/*
 * .-'_.---._'-.
 * ||####|(__)||   Protect your secrets, protect your business.
 *   \\()|##//       Secure your sensitive data with Aegis.
 *    \\ |#//                  <aegis.z2h.dev>
 *     .\_/.
 */

package sentry

import (
	"github.com/zerotohero-dev/aegis-sdk-go/internal/env"
	"log"
	"time"
)

// Watch synchronizes the internal state of the sidecar by talking to
// Aegis Safe regularly. It periodically calls Fetch behind-the-scenes to
// get its work done. Once it fetches the secrets, it saves it to
// the location defined in the `AEGIS_SIDECAR_SECRETS_PATH` environment
// variable (`/opt/aegis/secrets.json` by default).
func Watch() {
	ticker := time.NewTicker(env.SentryPollInterval())
	for {
		select {
		case <-ticker.C:
			err := fetchSecrets()
			if err != nil {
				log.Println("Could not fetch secrets", err.Error(), ". Will retry.")
				return
			}
		}
	}
}
