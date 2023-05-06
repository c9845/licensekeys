package middleware

import (
	"net/http"
	"strings"
)

// This file sets headers for security purposes to protect against cross site scripts,
// clickjacking, etc.
//
// The SecHeaders func should be called upon every page load.

// SecHeaders sets http headers for security purposes.
func SecHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("strict-transport-security", "max-age=60")
		w.Header().Set("x-frame-options", "sameorigin")
		w.Header().Set("x-xss-protection", "1; mode=block")
		w.Header().Set("x-content-type-options", "nosniff")

		//Content Security Policy
		//Any resources used must fit into one of these groups to be whitelisted.
		//
		//Try to serve all third-party stuff from the same CDN just to limit the amount
		//of different hosts we have to list here.
		defaultSrc := []string{
			"'self'",
			"https://cdnjs.cloudflare.com/", //bootstrap popper.js, chart.js, moment.js (for charts), remove this when browsers support prefetch-src
			";",
		}
		connectSrc := []string{
			"'self'",
			";",
		}
		scriptSrc := []string{
			"'self'",
			"'unsafe-inline'",               //bootstrap tooltip is only initialized when needed and is injected as a <script> tag
			"'unsafe-eval'",                 //vue needs this for templating
			"https://cdnjs.cloudflare.com/", //vue, bootstrap, bootstrap popper.js, chart.js, moment.js (for charts)
			";",
		}
		styleSrc := []string{
			"'self'",
			"'unsafe-inline'",               //inline styles
			"https://cdnjs.cloudflare.com/", //bootstrap, font awesome icons
			";",
		}
		fontSrc := []string{
			"'self'",
			"https://cdnjs.cloudflare.com/", //font awesome icons
			";",
		}
		imgSrc := []string{
			"'self'",
			"data:", //for embedding images in printable docs, 2FA QR code.
			";",
		}

		csp := "" +
			" default-src " + strings.Join(defaultSrc, " ") +
			" connect-src " + strings.Join(connectSrc, " ") +
			" script-src " + strings.Join(scriptSrc, " ") +
			" style-src " + strings.Join(styleSrc, " ") +
			" font-src " + strings.Join(fontSrc, " ") +
			" img-src " + strings.Join(imgSrc, " ")
		//" prefetch-src " + strings.Join(prefetchSrc, " ")
		w.Header().Set("content-security-policy", csp)

		//move to next middleware or handler
		next.ServeHTTP(w, r)
	})
}
