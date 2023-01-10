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
	"bytes"
	"context"
	"encoding/json"
	"github.com/pkg/errors"
	"github.com/spiffe/go-spiffe/v2/spiffeid"
	"github.com/spiffe/go-spiffe/v2/spiffetls/tlsconfig"
	"github.com/spiffe/go-spiffe/v2/workloadapi"
	"github.com/zerotohero-dev/aegis-core/entity/reqres/v1"
	"github.com/zerotohero-dev/aegis-core/validation"
	"github.com/zerotohero-dev/aegis-sdk-go/env"
	"io"
	"log"
	"net/http"
	"net/url"
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

func Fetch() (string, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	source, err := workloadapi.NewX509Source(
		ctx, workloadapi.WithClientOptions(workloadapi.WithAddr(env.SpiffeSocketUrl())),
	)

	if err != nil {
		return "", errors.Wrap(err, "failed getting SVID Bundle from the SPIRE Workload API")
	}

	svid, err := source.GetX509SVID()
	if err != nil {
		return "", errors.Wrap(err, "malformed SVID")
	}

	defer func(source *workloadapi.X509Source) {
		if source == nil {
			return
		}
		err := source.Close()
		if err != nil {
			log.Println("Problem closing the workload source.")
		}
	}(source)

	// Make sure that we are calling Safe from a workload that Aegis knows about.
	if !validation.IsWorkload(svid.ID.String()) {
		log.Fatalf("Untrusted workload. Killing the container.")
	}

	authorizer := tlsconfig.AdaptMatcher(func(id spiffeid.ID) error {
		if validation.IsSafe(id.String()) {
			return nil
		}

		return errors.New("I don’t know you, and it’s crazy: '" + id.String() + "'")
	})

	p, err := url.JoinPath(env.SafeEndpointUrl(), "/v1/fetch")
	if err != nil {
		log.Fatalf("Problem generating server url. Killing the container.")
	}

	tlsConfig := tlsconfig.MTLSClientConfig(source, source, authorizer)
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
	}

	sr := v1.SecretFetchRequest{}

	md, err := json.Marshal(sr)
	if err != nil {
		return "", errors.Wrap(err, "trouble generating payload")
	}

	r, err := client.Post(p, "application/json", bytes.NewBuffer(md))
	if err != nil {
		return "", errors.Wrap(err, "problem connecting to Aegis Safe API endpoint")
	}

	defer func(b io.ReadCloser) {
		if b == nil {
			return
		}
		err2 := b.Close()
		if err2 != nil {
			log.Println("Problem closing response body.")
		}
	}(r.Body)

	body, err := io.ReadAll(r.Body)
	if err != nil {
		return "", errors.Wrap(err, "unable to read the response body from Aegis Safe API endpoint")
	}

	var sfr v1.SecretFetchResponse

	err = json.Unmarshal(body, &sfr)
	if err != nil {
		return "", errors.Wrap(err, "unable to deserialize response")
	}

	data := sfr.Data
	return data, nil
}

func fetchSecrets() error {
	data, _ := Fetch()
	return saveData(data)
}
