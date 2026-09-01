package surface

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

//go:embed manifest.json
var manifestBytes []byte

type Manifest struct {
	SchemaVersion             int            `json:"schema_version"`
	ArtifactType              string         `json:"artifact_type"`
	SharedCloudResponseSchema map[string]any `json:"shared_cloud_response_schema"`
	Domains                   []Domain       `json:"domains"`
	Capabilities              []Capability   `json:"capabilities"`
	SupportTools              []SupportTool  `json:"support_tools,omitempty"`
}

type Domain struct {
	Name            string `json:"name"`
	Description     string `json:"description"`
	CapabilityCount int    `json:"capability_count"`
}

type Capability struct {
	Name              string         `json:"name"`
	DisplayName       string         `json:"display_name"`
	Domain            string         `json:"domain"`
	Description       string         `json:"description"`
	Cloud             bool           `json:"cloud"`
	Local             bool           `json:"local"`
	Async             bool           `json:"async"`
	Method            string         `json:"method"`
	Path              string         `json:"path"`
	Parameters        map[string]any `json:"parameters"`
	Response          map[string]any `json:"response"`
	LocalDependencies []string       `json:"local_dependencies,omitempty"`
	LocalInputSchema  map[string]any `json:"local_input_schema,omitempty"`
	LocalOutputSchema map[string]any `json:"local_output_schema,omitempty"`
}

type SupportTool struct {
	Name              string         `json:"name"`
	Domain            string         `json:"domain"`
	DomainDescription string         `json:"domain_description"`
	Cloud             bool           `json:"cloud"`
	Local             bool           `json:"local"`
	InputSchema       map[string]any `json:"input_schema"`
	OutputSchema      map[string]any `json:"output_schema"`
	LocalDependencies []string       `json:"local_dependencies,omitempty"`
}

var (
	currentManifest  Manifest
	capabilityIndex  map[string]Capability
	supportToolIndex map[string]SupportTool
)

func init() {
	if err := json.Unmarshal(manifestBytes, &currentManifest); err != nil {
		panic(fmt.Sprintf("invalid embedded surface manifest: %v", err))
	}
	if currentManifest.SchemaVersion != 1 ||
		currentManifest.ArtifactType != "cli_runtime_surface" {
		panic("invalid embedded surface manifest identity")
	}
	if err := validateSharedCloudResponse(currentManifest.SharedCloudResponseSchema); err != nil {
		panic("invalid shared Cloud response contract: " + err.Error())
	}
	capabilityIndex = make(map[string]Capability, len(currentManifest.Capabilities))
	supportToolIndex = make(map[string]SupportTool, len(currentManifest.SupportTools))
	domainCounts := map[string]int{}
	for _, capability := range currentManifest.Capabilities {
		if capability.Name == "" || capability.Domain == "" ||
			capability.Description == "" || capability.Method == "" ||
			capability.Path == "" || !capability.Cloud {
			panic("invalid capability in embedded surface manifest")
		}
		if _, exists := capabilityIndex[capability.Name]; exists {
			panic("duplicate capability in embedded surface manifest: " + capability.Name)
		}
		capabilityIndex[capability.Name] = capability
		domainCounts[capability.Domain]++
	}
	for _, support := range currentManifest.SupportTools {
		if support.Name == "" || support.Domain == "" ||
			support.Cloud || !support.Local ||
			len(support.InputSchema) == 0 || len(support.OutputSchema) == 0 {
			panic("invalid support tool in embedded surface manifest")
		}
		if _, exists := capabilityIndex[support.Name]; exists {
			panic("support tool collides with capability: " + support.Name)
		}
		if _, exists := supportToolIndex[support.Name]; exists {
			panic("duplicate support tool in embedded surface manifest: " + support.Name)
		}
		supportToolIndex[support.Name] = support
	}
	for _, domain := range currentManifest.Domains {
		if domain.CapabilityCount != domainCounts[domain.Name] {
			panic("domain count mismatch in embedded surface manifest: " + domain.Name)
		}
		delete(domainCounts, domain.Name)
	}
	if len(domainCounts) != 0 {
		names := make([]string, 0, len(domainCounts))
		for name := range domainCounts {
			names = append(names, name)
		}
		sort.Strings(names)
		panic("missing domains in embedded surface manifest: " + fmt.Sprint(names))
	}
}

func validateSharedCloudResponse(schema map[string]any) error {
	if schema["type"] != "object" {
		return fmt.Errorf("output type must be object")
	}
	properties, ok := schema["properties"].(map[string]any)
	if !ok || len(properties) != 1 {
		return fmt.Errorf("output properties must contain only usage")
	}
	usage, ok := properties["usage"].(map[string]any)
	if !ok || usage["type"] != "object" {
		return fmt.Errorf("usage type must be object")
	}
	if additional, ok := usage["additionalProperties"].(bool); !ok || additional {
		return fmt.Errorf("usage additionalProperties must be false")
	}
	if strings.TrimSpace(fmt.Sprint(usage["description"])) == "" {
		return fmt.Errorf("usage description is required")
	}
	usageProperties, ok := usage["properties"].(map[string]any)
	if !ok || len(usageProperties) == 0 {
		return fmt.Errorf("usage properties must be a non-empty object")
	}
	required, ok := usage["required"].([]any)
	if !ok || len(required) == 0 || len(required) != len(usageProperties) {
		return fmt.Errorf("usage required must be a non-empty array")
	}
	seen := make(map[string]struct{}, len(required))
	for index, item := range required {
		name, ok := item.(string)
		if !ok || strings.TrimSpace(name) == "" {
			return fmt.Errorf("usage required[%d] must be a non-empty string", index)
		}
		if _, duplicate := seen[name]; duplicate {
			return fmt.Errorf("usage required field is duplicated: %s", name)
		}
		seen[name] = struct{}{}
		field, ok := usageProperties[name].(map[string]any)
		if !ok {
			return fmt.Errorf("usage required field is missing: %s", name)
		}
		fieldType, ok := field["type"].(string)
		if !ok || strings.TrimSpace(fieldType) == "" {
			return fmt.Errorf("usage.%s type is required", name)
		}
		if strings.TrimSpace(fmt.Sprint(field["description"])) == "" {
			return fmt.Errorf("usage.%s description is required", name)
		}
	}
	return nil
}

// SharedCloudUsage validates a service-provided usage value against the
// source-locked shared schema embedded in the runtime manifest. It deliberately
// does not calculate, infer, rename, flatten, or fill any field.
func SharedCloudUsage(value any) (map[string]any, bool) {
	usageSchema, ok := currentManifest.SharedCloudResponseSchema["properties"].(map[string]any)
	if !ok {
		return nil, false
	}
	schema, ok := usageSchema["usage"].(map[string]any)
	if !ok {
		return nil, false
	}
	valueObject, ok := value.(map[string]any)
	if !ok {
		return nil, false
	}
	properties, ok := schema["properties"].(map[string]any)
	if !ok || len(valueObject) != len(properties) {
		return nil, false
	}
	required, ok := schema["required"].([]any)
	if !ok || len(required) != len(properties) {
		return nil, false
	}
	for _, item := range required {
		name, ok := item.(string)
		if !ok {
			return nil, false
		}
		fieldSchema, ok := properties[name].(map[string]any)
		if !ok {
			return nil, false
		}
		fieldValue, exists := valueObject[name]
		if !exists || !matchesJSONType(fieldSchema["type"], fieldValue) {
			return nil, false
		}
	}
	return valueObject, true
}

func matchesJSONType(schemaType any, value any) bool {
	switch schemaType {
	case "number":
		switch value.(type) {
		case float64, float32, int, int8, int16, int32, int64,
			uint, uint8, uint16, uint32, uint64, json.Number:
			return true
		default:
			return false
		}
	case "integer":
		switch value.(type) {
		case int, int8, int16, int32, int64,
			uint, uint8, uint16, uint32, uint64:
			return true
		default:
			return false
		}
	case "string":
		_, ok := value.(string)
		return ok
	case "boolean":
		_, ok := value.(bool)
		return ok
	case "object":
		_, ok := value.(map[string]any)
		return ok
	case "array":
		_, ok := value.([]any)
		return ok
	default:
		return false
	}
}

func Current() Manifest {
	return currentManifest
}

func Lookup(name string) (Capability, bool) {
	capability, ok := capabilityIndex[name]
	return capability, ok
}

func LookupSupportTool(name string) (SupportTool, bool) {
	support, ok := supportToolIndex[name]
	return support, ok
}
