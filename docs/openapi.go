// Package docs embeds published BeeBuzz API documentation assets.
package docs

import _ "embed"

// OpenAPIYAML is the canonical OpenAPI contract published by the server.
//
//go:embed openapi.yaml
var OpenAPIYAML []byte
