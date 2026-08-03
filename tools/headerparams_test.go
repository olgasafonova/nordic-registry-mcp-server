package tools

import (
	"encoding/json"
	"strings"
	"testing"
)

type headerParamsTestArgs struct {
	OrgNumber string `json:"org_number" jsonschema:"9-digit organization number"`
	Full      bool   `json:"full,omitempty" jsonschema:"Return the full record"`
	Filter    struct {
		Kind string `json:"kind,omitempty"`
	} `json:"filter,omitempty"`
}

// TestHeaderAnnotatedSchemaAttachesAnnotation: the returned schema carries the
// x-mcp-header keyword on the named property, in the exact shape the SDK's
// collectParamHeaderAnnotations reads (a plain string on the property object).
func TestHeaderAnnotatedSchemaAttachesAnnotation(t *testing.T) {
	spec := ToolSpec{
		Name:         "test_tool",
		HeaderParams: map[string]string{"org_number": "Org-Number"},
	}

	schema := headerAnnotatedSchema[headerParamsTestArgs](spec)

	raw, err := json.Marshal(schema)
	if err != nil {
		t.Fatalf("marshaling schema: %v", err)
	}
	var decoded struct {
		Properties map[string]map[string]any `json:"properties"`
	}
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("unmarshaling schema: %v", err)
	}

	got, ok := decoded.Properties["org_number"]["x-mcp-header"]
	if !ok {
		t.Fatalf("org_number property has no x-mcp-header keyword:\n%s", raw)
	}
	if got != "Org-Number" {
		t.Errorf("x-mcp-header = %v, want Org-Number", got)
	}

	// Un-annotated properties must stay clean.
	if _, ok := decoded.Properties["full"]["x-mcp-header"]; ok {
		t.Error("full property unexpectedly carries x-mcp-header")
	}

	// The rest of the inferred schema must survive: descriptions and types.
	if decoded.Properties["org_number"]["description"] != "9-digit organization number" {
		t.Errorf("org_number description = %v, want the jsonschema tag value", decoded.Properties["org_number"]["description"])
	}
}

// TestHeaderAnnotatedSchemaPanicsOnUnknownProperty: a HeaderParams entry that
// names a property the Args struct does not have is a programming error and
// must fail loudly at registration, because the SDK silently ignores
// malformed annotations.
func TestHeaderAnnotatedSchemaPanicsOnUnknownProperty(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic for unknown property")
		}
		if !strings.Contains(r.(string), "unknown property") {
			t.Errorf("panic %v does not mention unknown property", r)
		}
	}()
	headerAnnotatedSchema[headerParamsTestArgs](ToolSpec{
		Name:         "test_tool",
		HeaderParams: map[string]string{"nope": "Nope"},
	})
}

// TestHeaderAnnotatedSchemaPanicsOnNonPrimitive: x-mcp-header is only valid
// on string/integer/boolean properties; an object property must be rejected
// at registration rather than dropped from tools/list at serve time.
func TestHeaderAnnotatedSchemaPanicsOnNonPrimitive(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic for non-primitive property")
		}
		if !strings.Contains(r.(string), "only string, integer, boolean") {
			t.Errorf("panic %v does not mention the allowed types", r)
		}
	}()
	headerAnnotatedSchema[headerParamsTestArgs](ToolSpec{
		Name:         "test_tool",
		HeaderParams: map[string]string{"filter": "Filter"},
	})
}

// TestAllToolsHeaderParamsAreWellFormed pins which tools declare header
// passthrough, so an accidental addition or removal fails here, and reminds
// the author that every declared suffix must also be in the CORS allowlist
// (security.go Access-Control-Allow-Headers).
func TestAllToolsHeaderParamsAreWellFormed(t *testing.T) {
	want := map[string]map[string]string{
		"norway_get_company": {"org_number": "Org-Number"},
	}

	got := map[string]map[string]string{}
	for _, spec := range AllTools {
		if len(spec.HeaderParams) > 0 {
			got[spec.Name] = spec.HeaderParams
		}
	}

	if len(got) != len(want) {
		t.Errorf("tools with HeaderParams = %v, want %v", got, want)
	}
	for name, params := range want {
		for prop, header := range params {
			if got[name][prop] != header {
				t.Errorf("%s HeaderParams[%s] = %q, want %q", name, prop, got[name][prop], header)
			}
		}
	}
}
