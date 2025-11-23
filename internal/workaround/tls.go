// Package workaround exists to avoid TLS errors from Stoat
package workaround

import "net/http"

// Do is a workaround for cloudflare issuing invalid certificates... pls fix >:(.
func Do(req *http.Request) (*http.Response, error) {
	transport := http.DefaultTransport.(*http.Transport).Clone() //nolint:forcetypeassert
	transport.TLSClientConfig.InsecureSkipVerify = req.Host == "cdn.stoatusercontent.com"

	return (&http.Client{Transport: transport}).Do(req) //nolint:wrapcheck
}
