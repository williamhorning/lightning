package discord

import "net/http"

type rewriteTransport struct {
	apiHost string
	cdnHost string
}

func (t *rewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	cloned := req.Clone(req.Context())

	switch cloned.URL.Host {
	case "discord.com":
		cloned.URL.Host = t.apiHost
		cloned.Host = t.apiHost + ":443"
	case "cdn.discordapp.com":
		cloned.URL.Host = t.cdnHost
		cloned.Host = t.cdnHost + ":443"
	default:
	}

	return http.DefaultTransport.RoundTrip(cloned) //nolint:wrapcheck
}
