/*
 * .-'_.---._'-.
 * ||####|(__)||   Protect your secrets, protect your business.
 *   \\()|##//       Secure your sensitive data with Aegis.
 *    \\ |#//                  <aegis.z2h.dev>
 *     .\_/.
 */

package startup

import (
	"github.com/zerotohero-dev/aegis-core/env"
	"github.com/zerotohero-dev/aegis-sdk-go/sentry"
	"os"
	"time"
)

func initialized() bool {
	r, _ := sentry.Fetch()
	v := r.Data
	return v != ""
}

// Watch continuously polls the associated secret of the workload to exist.
// If the secret exists, and it is not empty, the function exits the init
// container with a success status code (0).
func Watch() {
	interval := env.InitContainerPollInterval()
	ticker := time.NewTicker(interval)
	for {
		select {
		case <-ticker.C:
			if initialized() {
				os.Exit(0)
			}
		}
	}
}
