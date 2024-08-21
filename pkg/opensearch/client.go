package opensearch

import (
	"crypto/tls"
	"net/http"
	"os"

	"github.com/opensearch-project/opensearch-go"
)

func NewClientConfig() opensearch.Config {
	insecureSkipVerify := os.Getenv("OPENSEARCH_TLS_INSECURE") != ""
	address := os.Getenv("OPENSEARCH_URL")
	user := os.Getenv("OPENSEARCH_USER")
	pass := os.Getenv("OPENSEARCH_PASS")

	return opensearch.Config{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: insecureSkipVerify},
		},
		Addresses: []string{address},
		Username:  user,
		Password:  pass,
	}
}
