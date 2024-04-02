# Search GH

## Workflows

```shell
go run . workflow runs > out.json
split -l 250 out.json split
find . -name "split*" -exec \   
    curl -XPUT --data-binary @{} --insecure -H "Content-Type: application/json" -u \
        admin:$OPENSEARCH_INITIAL_ADMIN_PASSWORD https://localhost:9200/_bulk --verbose \;
```