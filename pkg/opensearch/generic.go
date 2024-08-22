package opensearch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"

	opensearchgo "github.com/opensearch-project/opensearch-go"
	"github.com/opensearch-project/opensearch-go/opensearchapi"
)

func doGenericRequest(ctx context.Context, client *opensearchgo.Client, req opensearchapi.Request) (map[string]any, error) {
	resp, err := req.Do(ctx, client)
	if err != nil {
		return nil, fmt.Errorf("unexpected error sending response to OpenSearch: %w", err)
	}
	defer resp.Body.Close()

	bodyBuf := &bytes.Buffer{}

	if _, err = io.Copy(bodyBuf, resp.Body); err != nil {
		return nil, fmt.Errorf("unexpected error while reading response from OpenSearch: %w", err)
	}

	body := bodyBuf.Bytes()

	if resp.IsError() {
		return nil, fmt.Errorf("unexpected error in response from OpenSearch: %s", body)
	}

	unstructured := map[string]any{}
	if err := json.Unmarshal(body, &unstructured); err != nil {
		return nil, fmt.Errorf("unable to parse response from OpenSearch: %w", err)
	}

	return unstructured, nil
}
