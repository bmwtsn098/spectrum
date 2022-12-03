package openapi3edit

import (
	"errors"
	"fmt"
	"strings"

	oas3 "github.com/getkin/kin-openapi/openapi3"
	"github.com/grokify/mogo/net/http/pathmethod"
	"github.com/grokify/mogo/type/maputil"
	"github.com/grokify/spectrum/openapi3"
)

const (
	SecuritySchemeApikeyDefaultName      = "ApiKeyAuth" // #nosec G101
	SecuritySchemeBearertokenDefaultName = "BearerAuth"
	SchemeHTTP                           = "http"
	TokenTypeBearer                      = "bearer"
)

// SecuritySchemeAddBearertoken adds bearer token auth
// to spec and operations.
func SecuritySchemeAddBearertoken(spec *openapi3.Spec, schemeName, bearerFormat string, inclTags, skipTags []string) {
	schemeName = strings.TrimSpace(schemeName)
	if len(schemeName) == 0 {
		schemeName = SecuritySchemeBearertokenDefaultName
	}
	SecuritySchemeBearertokenAddDefinition(spec, schemeName, bearerFormat)
	SecuritySchemeBearertokenAddOperationsByTags(spec, schemeName, inclTags, skipTags)
}

func SecuritySchemeBearertokenAddOperationsByTags(spec *openapi3.Spec, schemeName string, inclTags, skipTags []string) {
	inclTagsMap := map[string]int{}
	for _, tag := range inclTags {
		tag = strings.ToLower(strings.TrimSpace(tag))
		if len(tag) > 0 {
			inclTagsMap[tag] = 1
		}
	}
	skipTagsMap := map[string]int{}
	for _, tag := range skipTags {
		tag = strings.ToLower(strings.TrimSpace(tag))
		if len(tag) > 0 {
			skipTagsMap[tag] = 1
		}
	}
	openapi3.VisitOperations(spec, func(skipPath string, skipMethod string, op *oas3.Operation) {
		addSecurity := false
		for _, tag := range op.Tags {
			tag = strings.ToLower(strings.TrimSpace(tag))
			if _, ok := inclTagsMap[tag]; ok {
				addSecurity = true
			}
			if len(inclTagsMap) == 0 {
				if len(skipTagsMap) == 0 {
					addSecurity = true
				} else if _, ok := skipTagsMap[tag]; !ok {
					addSecurity = true
				}
			}
		}
		if addSecurity {
			ope := NewOperationEdit(skipPath, skipMethod, op)
			ope.AddSecurityRequirement(schemeName, []string{})
		}
	})
}

func SecuritySchemeBearertokenAddDefinition(spec *openapi3.Spec, schemeName, bearerFormat string) {
	schemeName = strings.TrimSpace(schemeName)
	bearerFormat = strings.TrimSpace(bearerFormat)
	if len(schemeName) == 0 {
		schemeName = SecuritySchemeBearertokenDefaultName
	}
	scheme := &oas3.SecuritySchemeRef{
		Value: &oas3.SecurityScheme{
			Type:   SchemeHTTP,
			Scheme: TokenTypeBearer}}
	if len(bearerFormat) > 0 {
		scheme.Value.BearerFormat = bearerFormat
	}
	spec.Components.SecuritySchemes[schemeName] = scheme
}

// AddAPIKey adds an API Key definition to the spec.
// https://swagger.io/docs/specification/authentication/api-keys/
func SecuritySchemeApikeyAddDefinition(spec *openapi3.Spec, schemeName, location, name string) error {
	schemeName = strings.TrimSpace(schemeName)
	location = strings.TrimSpace(location)
	name = strings.TrimSpace(name)
	if len(schemeName) == 0 {
		schemeName = SecuritySchemeApikeyDefaultName
	}
	if len(location) == 0 {
		return errors.New("api key security scheme location cannot be empty. Must be one of: [\"header\", \"query\", \"cookie\"]")
	} else if location != "header" && location != "query" && location != "cookie" {
		return fmt.Errorf("api key security scheme invalid location [%s], must be one of: [\"header\", \"query\", \"cookie\"]", location)
	}
	if len(name) == 0 {
		return errors.New("api key security scheme name cannot be empty")
	}
	if spec.Components.SecuritySchemes == nil {
		spec.Components.SecuritySchemes = map[string]*oas3.SecuritySchemeRef{}
	}
	spec.Components.SecuritySchemes[schemeName] = &oas3.SecuritySchemeRef{
		Value: &oas3.SecurityScheme{
			Type: "apiKey",
			In:   location,
			Name: name,
		},
	}
	return nil
}

func SecuritySchemeApikeyAddOperations(spec *openapi3.Spec, tags []string, keyName string) {
	keyName = strings.TrimSpace(keyName)
	if len(keyName) == 0 {
		keyName = SecuritySchemeApikeyDefaultName
	}
	tagsMap := map[string]int{}
	for _, tagName := range tags {
		tagName = strings.TrimSpace(tagName)
		tagsMap[tagName] = 1
	}
	openapi3.VisitOperations(spec, func(skipPath, skipMethod string, op *oas3.Operation) {
		if !maputil.StringKeysExist(tagsMap, op.Tags, false) {
			return
		}
		ope := NewOperationEdit(skipPath, skipMethod, op)
		ope.AddSecurityRequirement(keyName, []string{})
		//SecuritySchemeAddOperation(op, keyName, []string{})
	})
}

/*
func MapSliceIntersection(haystack map[string]int, needles []string, unique bool) []string {
	if unique {
		needles = stringsutil.SliceCondenseSpace(needles, true, false)
	}
	matches := []string{}
	for _, needle := range needles {
		if _, ok := haystack[needle]; ok {
			matches = append(matches, needle)
		}
	}
	return matches
}
*/

// RemoveOperationsSecurity removes the security property
// for all operations. It is useful when building a spec
// to get individual specs to validate before setting the
// correct security property.
func RemoveOperationsSecurity(spec *openapi3.Spec, inclPathMethods []string) error {
	pms := pathmethod.NewPathMethodSet()
	err := pms.Add(inclPathMethods...)
	if err != nil {
		return err
	}
	countInclPathMethods := pms.Count()
	openapi3.VisitOperations(spec, func(opPath, opMethod string, op *oas3.Operation) {
		if countInclPathMethods > 0 && !pms.Exists(opPath, opMethod) {
			return
		}
		op.Security = &oas3.SecurityRequirements{}
	})
	return nil
}
