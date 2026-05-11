// preprocess splits a raw upstream OpenAPI spec into the two-file layout used
// by this project (openapi.yaml for paths, components.openapi.yaml for schemas)
// and applies mechanical transformations that make the spec ready for overlay-
// based code generation.
//
// Transformations applied:
//   - $ref retarget: #/components/… → components.openapi.yaml#/components/…
//   - Content type normalization: */* → application/json
//   - Error response stripping: removes 400/401/403/500 (overlay adds default)
//   - Security injection: ensures security: [{bearerAuth: []}] is present
//
// Usage:
//
//	go run ./pkg/internal/openapi/preprocess -spec path/to/service-api.json [-out output-dir]
package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"gopkg.in/yaml.v3"
)

// document controls the top-level key order for the output files. Each output
// populates either Paths or Components, relying on omitempty to suppress the other.
type document struct {
	OpenAPI    string `yaml:"openapi"`
	Info       any    `yaml:"info"`
	Servers    any    `yaml:"servers,omitempty"`
	Security   any    `yaml:"security,omitempty"`
	Tags       any    `yaml:"tags,omitempty"`
	Paths      any    `yaml:"paths,omitempty"`
	Components any    `yaml:"components,omitempty"`
}

func main() {
	specPath := flag.String("spec", "", "path to raw upstream OpenAPI spec (JSON or YAML)")
	outDirFlag := flag.String(
		"out",
		"",
		"output directory for openapi.yaml and components.openapi.yaml (omit to derive from this program's source location)",
	)

	flag.Parse()

	if *specPath == "" {
		log.Fatal("-spec flag is required")
	}

	runErr := runPreprocess(*specPath, *outDirFlag)
	if runErr != nil {
		log.Fatal(runErr)
	}
}

func runPreprocess(specPath, outDirFlag string) error {
	//nolint:gosec // G304: path comes from explicit -spec CLI flag
	data, err := os.ReadFile(specPath)
	if err != nil {
		return fmt.Errorf("reading spec: %w", err)
	}

	raw := parseSpec(data)

	paths := raw["paths"]
	components := raw["components"]

	retargetRefs(paths)
	retargetRefs(components)
	normalizeContentTypes(paths)
	stripErrorResponses(paths)
	fixCommaSeparatedEnums(components)
	normalizeSecurityRefs(paths)

	security := raw["security"]
	if security == nil {
		security = []any{map[string]any{"bearerAuth": []any{}}}
	}

	openapi, ok := raw["openapi"].(string)
	if !ok {
		return errors.New("spec is missing the openapi version field")
	}

	outDir := resolveOutputDir(outDirFlag)

	base := document{
		OpenAPI:  openapi,
		Info:     raw["info"],
		Servers:  raw["servers"],
		Security: security,
		Tags:     raw["tags"],
	}

	pathsDoc := base
	pathsDoc.Paths = paths

	componentsDoc := base
	componentsDoc.Components = components

	marshalToFile(filepath.Join(outDir, "openapi.yaml"), pathsDoc)
	marshalToFile(filepath.Join(outDir, "components.openapi.yaml"), componentsDoc)

	log.Printf("wrote openapi.yaml and components.openapi.yaml to %s", outDir)

	return nil
}

func resolveOutputDir(outDirFlag string) string {
	if outDirFlag != "" {
		return filepath.Clean(outDirFlag)
	}

	return scriptDir()
}

// parseSpec detects the input format (JSON or YAML) and parses it into a generic map.
func parseSpec(data []byte) map[string]any {
	var raw map[string]any

	trimmed := bytes.TrimSpace(data)
	if len(trimmed) > 0 && trimmed[0] == '{' {
		err := json.Unmarshal(data, &raw)
		if err != nil {
			log.Fatalf("parsing JSON: %v", err)
		}
	} else {
		err := yaml.Unmarshal(data, &raw)
		if err != nil {
			log.Fatalf("parsing YAML: %v", err)
		}
	}

	return raw
}

// walk recursively traverses the parsed OpenAPI tree, calling visit on every map node.
func walk(node any, visit func(map[string]any)) {
	switch value := node.(type) {
	case map[string]any:
		visit(value)

		for _, child := range value {
			walk(child, visit)
		}
	case []any:
		for _, child := range value {
			walk(child, visit)
		}
	}
}

// retargetRefs recursively rewrites $ref values from internal (#/components/…) to
// external (components.openapi.yaml#/components/…) format.
func retargetRefs(node any) {
	walk(node, func(value map[string]any) {
		if ref, ok := value["$ref"].(string); ok && strings.HasPrefix(ref, "#/components/") {
			value["$ref"] = "components.openapi.yaml" + ref
		}
	})
}

// normalizeContentTypes recursively replaces */* content type keys with application/json.
func normalizeContentTypes(node any) {
	walk(node, func(value map[string]any) {
		if wildcardValue, ok := value["*/*"]; ok {
			if _, hasJSON := value["application/json"]; !hasJSON {
				value["application/json"] = wildcardValue
			}

			delete(value, "*/*")
		}
	})
}

// eachOperation iterates over all HTTP operations in the paths object, calling
// handleOperation with the operation map for each one.
func eachOperation(paths any, handleOperation func(operation map[string]any)) {
	pathsMap, ok := paths.(map[string]any)
	if !ok {
		return
	}

	for _, pathItem := range pathsMap {
		itemMap, ok := pathItem.(map[string]any)
		if !ok {
			continue
		}

		for _, method := range []string{"get", "post", "put", "delete", "patch"} {
			if operation, ok := itemMap[method].(map[string]any); ok {
				handleOperation(operation)
			}
		}
	}
}

// stripErrorResponses removes explicit error status codes (400, 401, 403, 500) from all
// operation responses. The overlay adds a unified default response with ErrorRS instead.
func stripErrorResponses(paths any) {
	eachOperation(paths, func(operation map[string]any) {
		if responses, ok := operation["responses"].(map[string]any); ok {
			for _, status := range []string{"400", "401", "403", "500"} {
				delete(responses, status)
			}
		}
	})
}

// fixCommaSeparatedEnums fixes malformed enum definitions where multiple values are
// packed into a single comma-separated string instead of being separate array items.
// It also relocates enums from array-typed properties to their items sub-schema,
// since oapi-codegen cannot generate valid Go for enums at the array level.
func fixCommaSeparatedEnums(node any) {
	walk(node, func(value map[string]any) {
		if enumValue, ok := value["enum"]; ok {
			splitCommaSeparatedEnum(value, enumValue)
			relocateArrayEnum(value)
		}
	})
}

// splitCommaSeparatedEnum expands a single comma-separated enum string into
// separate array elements.
func splitCommaSeparatedEnum(value map[string]any, enumValue any) {
	elements, ok := enumValue.([]any)
	if !ok || len(elements) != 1 {
		return
	}

	single, ok := elements[0].(string)
	if !ok || !strings.Contains(single, ",") {
		return
	}

	parts := strings.Split(single, ",")
	expanded := make([]any, len(parts))

	for i, part := range parts {
		expanded[i] = strings.TrimSpace(part)
	}

	value["enum"] = expanded
}

// relocateArrayEnum moves an enum from an array-typed property down to its
// items sub-schema, where oapi-codegen expects it.
func relocateArrayEnum(value map[string]any) {
	if value["type"] != "array" {
		return
	}

	items, ok := value["items"].(map[string]any)
	if !ok {
		log.Printf(
			"warning: array-typed property has enum but no items schema — skipping relocation",
		)

		return
	}

	items["enum"] = value["enum"]
	delete(value, "enum")
}

// normalizeSecurityRefs fixes inconsistent security scheme casing in operation-level
// security declarations (e.g. BearerAuth vs bearerAuth) to match the scheme definition.
func normalizeSecurityRefs(paths any) {
	eachOperation(paths, func(operation map[string]any) {
		securityList, ok := operation["security"].([]any)
		if !ok {
			return
		}

		for _, item := range securityList {
			securityItem, ok := item.(map[string]any)
			if !ok {
				continue
			}

			if scopes, exists := securityItem["BearerAuth"]; exists {
				delete(securityItem, "BearerAuth")
				securityItem["bearerAuth"] = scopes
			}
		}
	})
}

// scriptDir returns the parent of the directory containing this source file,
// which is the openapi spec directory. It is used only as a fallback when the
// -out flag is not specified. It relies on runtime.Caller(0), which embeds the
// source path at compile time, so this only works correctly when invoked via
// "go run" (not from a pre-built binary).
func scriptDir() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		log.Fatal("cannot determine script directory")
	}

	return filepath.Dir(filepath.Dir(file))
}

// marshalToFile serializes doc to YAML and writes it to path.
func marshalToFile(path string, doc any) {
	var buf bytes.Buffer

	encoder := yaml.NewEncoder(&buf)
	encoder.SetIndent(2)

	err := encoder.Encode(doc)
	if err != nil {
		log.Fatalf("marshaling YAML for %s: %v", path, err)
	}

	err = encoder.Close()
	if err != nil {
		log.Fatalf("closing encoder for %s: %v", path, err)
	}

	err = os.WriteFile(path, buf.Bytes(), 0o600)
	if err != nil {
		log.Fatalf("writing %s: %v", path, err)
	}
}
