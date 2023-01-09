package ruletagstylefirstuppercase

import (
	"strconv"

	"github.com/grokify/mogo/encoding/jsonpointer"
	"github.com/grokify/mogo/text/stringcase"
	"github.com/grokify/spectrum/openapi3"
	"github.com/grokify/spectrum/openapi3lint/lintutil"
)

type RuleTagStyleFirstUpperCase struct {
	name string
}

func NewRule() RuleTagStyleFirstUpperCase {
	return RuleTagStyleFirstUpperCase{
		name: lintutil.RulenameTagStyleFirstUpperCase}
}

func (rule RuleTagStyleFirstUpperCase) Name() string {
	return rule.name
}

func (rule RuleTagStyleFirstUpperCase) Scope() string {
	return lintutil.ScopeOperation
}

func (rule RuleTagStyleFirstUpperCase) ProcessOperation(spec *openapi3.Spec, op *openapi3.Operation, opPointer, path, method string) []lintutil.PolicyViolation {
	return nil
}

func (rule RuleTagStyleFirstUpperCase) ProcessSpec(spec *openapi3.Spec, pointerBase string) []lintutil.PolicyViolation {
	vios := []lintutil.PolicyViolation{}
	openapi3.VisitOperations(spec, func(path, method string, op *openapi3.Operation) {
		if op == nil {
			return
		}
		opLoc := jsonpointer.PointerSubEscapeAll(
			"%s#/paths/%s/%s/tags/",
			pointerBase,
			path,
			method)
		for i, tag := range op.Tags {
			if !stringcase.IsFirstAlphaUpper(tag) {
				vios = append(vios, lintutil.PolicyViolation{
					RuleName: rule.Name(),
					Location: opLoc + strconv.Itoa(i),
					Value:    tag})
			}
		}
	})
	for i, tag := range spec.Tags {
		jsLoc := jsonpointer.PointerSubEscapeAll(
			"%s#/tags/%d/name", pointerBase, i)
		if tag == nil {
			vios = append(vios, lintutil.PolicyViolation{
				RuleName: rule.Name(),
				Location: jsLoc,
				Value:    ""})
		} else if !stringcase.IsFirstAlphaUpper(tag.Name) {
			vios = append(vios, lintutil.PolicyViolation{
				RuleName: rule.Name(),
				Location: jsLoc,
				Value:    tag.Name})
		}
	}
	return vios
}
