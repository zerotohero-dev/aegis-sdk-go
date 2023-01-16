/*
 * .-'_.---._'-.
 * ||####|(__)||   Protect your secrets, protect your business.
 *   \\()|##//       Secure your sensitive data with Aegis.
 *    \\ |#//                  <aegis.z2h.dev>
 *     .\_/.
 */

package sentry

import (
	"bufio"
	"github.com/pkg/errors"
	"github.com/zerotohero-dev/aegis-core/env"
	"os"
)

func saveData(data string) error {
	path := env.SidecarSecretsPath()

	f, err := os.Create(path)
	if err != nil {
		return errors.New("error saving data")
	}

	w := bufio.NewWriter(f)
	_, err = w.WriteString(data)
	if err != nil {
		return errors.New("error saving data")
	}

	err = w.Flush()
	if err != nil {
		return errors.Wrap(err, "error flushing writer")
	}

	return nil
}

func fetchSecrets() error {
	data, _ := Fetch()
	return saveData(data)
}
