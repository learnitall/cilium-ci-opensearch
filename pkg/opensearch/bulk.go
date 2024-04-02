package opensearch

import (
	"io"
	"strings"
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
