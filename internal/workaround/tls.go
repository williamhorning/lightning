// Package workaround exists to avoid TLS errors from Stoat
package workaround

import "net/http"

// Client is a workaround for cloudflare issuing invalid certificates... pls fix >:(.
var Client *http.Client //nolint:gochecknoglobals

func init() { //nolint:gochecknoinits
	transport := http.DefaultTransport.(*http.Transport).Clone() //nolint:forcetypeassert
	transport.TLSClientConfig.InsecureSkipVerify = true

	Client = &http.Client{Transport: transport}
}
