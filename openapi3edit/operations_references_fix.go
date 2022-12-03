package openapi3edit

import (
	"fmt"
	"regexp"
	"strings"

	oas3 "github.com/getkin/kin-openapi/openapi3"
	"github.com/grokify/spectrum/openapi3"
)

const (
	oas2BasePathDefinitions       = "#/definitions/"
	oas3BasePathComponentsSchemas = "#/components/schemas/"
)

var rxOAS2RefDefinition = regexp.MustCompile(`^#/definitions/(.*)`)

func SpecOperationsFixResponseReferences(spec *openapi3.Spec) []*openapi3.OperationMeta {
	errorOperations := []*openapi3.OperationMeta{}
	openapi3.VisitOperations(spec, func(path, method string, op *oas3.Operation) {
		if op == nil {
			return
		}
		for resCode, resRef := range op.Responses {
			if strings.Index(resRef.Ref, oas2BasePathDefinitions) == 0 {
				resRef.Ref = strings.TrimSpace(resRef.Ref)
				m := rxOAS2RefDefinition.FindStringSubmatch(resRef.Ref)
				if len(m) > 0 {
					resRefOrig := resRef.Ref
					resRef.Ref = oas3BasePathComponentsSchemas + m[1]
					om := openapi3.OperationToMeta(path, method, op, []string{})
					om.MetaNotes = append(om.MetaNotes, fmt.Sprintf("E_BAD_RESPONSE_REF_OAS2_DEF [%s] type[%s]", resCode, resRefOrig))
					errorOperations = append(errorOperations, om)
				}
			}
		}
	})
	return errorOperations
}

func FixFile(input, output string, prefix, indent string, validateOutput bool) (*openapi3.Spec, []*openapi3.OperationMeta, error) {
	spec, err := openapi3.ReadFile(input, false)
	if err != nil {
		return spec, []*openapi3.OperationMeta{}, err
	}
	errs := SpecOperationsFixResponseReferences(spec)
	output = strings.TrimSpace(output)
	if len(output) > 0 {
		smore := openapi3.SpecMore{Spec: spec}
		return spec, errs, smore.WriteFileJSON(output, 0644, prefix, indent)
	}
	return spec, errs, err
}
