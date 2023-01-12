/*
 * .-'_.---._'-.
 * ||####|(__)||   Protect your secrets, protect your business.
 *   \\()|##//       Secure your sensitive data with Aegis.
 *    \\ |#//                  <aegis.z2h.dev>
 *     .\_/.
 */

package sentry

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/pkg/errors"
	"github.com/spiffe/go-spiffe/v2/spiffeid"
	"github.com/spiffe/go-spiffe/v2/spiffetls/tlsconfig"
	"github.com/spiffe/go-spiffe/v2/workloadapi"
	reqres "github.com/zerotohero-dev/aegis-core/entity/reqres/v1"
	"github.com/zerotohero-dev/aegis-core/validation"
	"github.com/zerotohero-dev/aegis-sdk-go/internal/env"
	"io"
	"log"
	"net/http"
	"net/url"
)

// Fetch fetches the up-to-date secret that has been registered to the workload.
//
//	secret, err := sentry.Fetch()
//
// In case of a problem, Fetch will return an empty string and an error explaining
// what went wrong.
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
		return "", errors.Wrap(err, "error getting SVID from source")
	}

	defer func() {
		err := source.Close()
		if err != nil {
			log.Println("Problem closing the workload source.")
		}
	}()

	// Make sure that we are calling Safe from a workload that Aegis knows about.
	if !validation.IsWorkload(svid.ID.String()) {
		return "", errors.New("untrusted workload")
	}

	authorizer := tlsconfig.AdaptMatcher(func(id spiffeid.ID) error {
		if validation.IsSafe(id.String()) {
			return nil
		}

		return errors.New("I don’t know you, and it’s crazy: '" + id.String() + "'")
	})

	p, err := url.JoinPath(env.SafeEndpointUrl(), "/v1/fetch")
	if err != nil {
		return "", errors.New("problem generating server url")
	}

	tlsConfig := tlsconfig.MTLSClientConfig(source, source, authorizer)
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
	}

	sr := reqres.SecretFetchRequest{}
	md, err := json.Marshal(sr)
	if err != nil {
		return "", errors.Wrap(err, "trouble generating payload")
	}

	r, err := client.Post(p, "application/json", bytes.NewBuffer(md))
	if err != nil {
		return "", errors.Wrap(err, "problem connecting to Aegis Safe API endpoint")
	}

	defer func() {
		err2 := r.Body.Close()
		if err2 != nil {
			log.Println("Problem closing response body.")
		}
	}()

	body, err := io.ReadAll(r.Body)
	if err != nil {
		return "", errors.Wrap(
			err, "unable to read the response body from Aegis Safe API endpoint",
		)
	}

	var sfr reqres.SecretFetchResponse
	err = json.Unmarshal(body, &sfr)
	if err != nil {
		return "", errors.Wrap(err, "unable to deserialize response")
	}

	return sfr.Data, nil
}
