package opensearch

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/learnitall/cilium-ci-opensearch/pkg/github"
)

type BulkEntry struct {
	Index string
	ID    string
	Verb  string
	Data  []byte
}

func (b *BulkEntry) Write(target io.Writer) {
	if b == nil {
		return
	}

	builder := strings.Builder{}
	builder.WriteString("{ \"")
	builder.WriteString(b.Verb)
	builder.WriteString("\" : { \"_index\": \"")
	builder.WriteString(b.Index)
	builder.WriteString("\", \"_id\": \"")
	builder.WriteString(b.ID)
	builder.WriteString("\" } }\n")

	target.Write([]byte(builder.String()))
	target.Write(b.Data)
	target.Write([]byte("\n"))
}

// GetDocumentID returns a unique document ID for the given object.
// Equal objects have the same ID.
func GetDocumentID(obj any) (string, error) {
	switch o := obj.(type) {
	case *github.WorkflowRun:
		return o.NodeID, nil
	case github.JobRun:
		return o.NodeID, nil
	case github.StepRun:
		return fmt.Sprintf("%s-%d", o.JobRun.NodeID, o.Number), nil
	case github.Testsuite:
		return fmt.Sprintf("%s-%s", o.WorkflowRun.NodeID, o.Name), nil
	case github.Testcase:
		return fmt.Sprintf("%s-%s-%s", o.WorkflowRun.NodeID, o.Testsuite.Name, o.Name), nil
	}

	return "", fmt.Errorf("unable to determine document ID for object '%v'", obj)
}

func BulkWriteObjects[T any](objs []T, index string, target io.Writer) error {
	for _, obj := range objs {
		d, err := json.Marshal(obj)
		if err != nil {
			return fmt.Errorf("unable to marshal obj '%v': %v", obj, err)
		}

		id, err := GetDocumentID(obj)
		if err != nil {
			return err
		}

		(&BulkEntry{
			Index: index,
			ID:    id,
			Verb:  "index",
			Data:  d,
		}).Write(target)
	}

	return nil
}
