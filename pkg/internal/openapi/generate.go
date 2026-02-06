// Package openapi contains the OpenAPI specifications, overlays, and code generation configuration for
// the Report Portal API. Run go generate in this directory to regenerate the API client and types.
package openapi

//go:generate go tool oapi-codegen -config api.config.yaml openapi.yaml
//go:generate go tool oapi-codegen -config types.config.yaml components.openapi.yaml
