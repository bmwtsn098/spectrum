package openapi2postman2

import (
	"encoding/json"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"sort"
	"strings"

	"github.com/grokify/mogo/net/httputilmore"
	"github.com/grokify/spectrum/openapi2"
	"github.com/grokify/spectrum/postman2"
	"github.com/grokify/spectrum/postman2/simple"
)

// Configuration is a Spectrum configuration that holds information on how
// to create the Postman 2.0 collection including overriding Swagger 2.0
// spec values.
type Configuration struct {
	PostmanURLBase     string            `json:"postmanURLBase,omitempty"`
	PostmanURLHostname string            `json:"postmanURLHostname,omitempty"`
	PostmanHeaders     []postman2.Header `json:"postmanHeaders,omitempty"`
}

// Converter is the struct that manages the conversion.
type Converter struct {
	Configuration Configuration
	Swagger       openapi2.Specification
}

// NewConverter instantiates a new converter.
func NewConverter(cfg Configuration) Converter {
	return Converter{Configuration: cfg}
}

// MergeConvert builds a Postman 2.0 spec using a base Postman 2.0 collection
// and a Swagger 2.0 spec.
func (conv *Converter) MergeConvert(swaggerFilepath string, pmanBaseFilepath string, pmanSpecFilepath string) error {
	swag, err := openapi2.ReadSwagger2SpecFile(swaggerFilepath)
	if err != nil {
		return err
	}

	pman, err := simple.ReadCanonicalCollection(pmanBaseFilepath)
	if err != nil {
		return err
	}
	pm := Merge(conv.Configuration, pman, swag)

	bytes, err := json.MarshalIndent(pm, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(pmanSpecFilepath, bytes, 0600)
}

// Convert builds a Postman 2.0 spec using a Swagger 2.0 spec.
func (conv *Converter) Convert(swaggerFilepath string, pmanSpecFilepath string) error {
	swag, err := openapi2.ReadSwagger2SpecFile(swaggerFilepath)
	if err != nil {
		return err
	}
	pm := Convert(conv.Configuration, swag)

	bytes, err := json.MarshalIndent(pm, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(pmanSpecFilepath, bytes, 0600)
}

// Convert creates a Postman 2.0 collection from a configuration and Swagger 2.0 spec
func Convert(cfg Configuration, swag openapi2.Specification) postman2.Collection {
	return Merge(cfg, postman2.Collection{}, swag)
}

// Merge creates a Postman 2.0 collection from a configuration, base Postman
// 2.0 collection and Swagger 2.0 spec
func Merge(cfg Configuration, pman postman2.Collection, swag openapi2.Specification) postman2.Collection {
	if len(pman.Info.Name) == 0 {
		pman.Info.Name = strings.TrimSpace(swag.Info.Title)
	}
	if len(pman.Info.Description) == 0 {
		pman.Info.Description = strings.TrimSpace(swag.Info.Description)
	}
	if len(pman.Info.Schema) == 0 {
		pman.Info.Schema = "https://schema.getpostman.com/json/collection/v2.0.0/collection.json"
	}

	urls := []string{}
	for u := range swag.Paths {
		urls = append(urls, u)
	}
	sort.Strings(urls)

	for _, u := range urls {
		path := swag.Paths[u]

		if path.HasMethodWithTag(http.MethodGet) {
			pman = postmanAddItemToFolder(pman,
				Swagger2PathToPostman2APIItem(cfg, swag, u, http.MethodGet, path.Get),
				strings.TrimSpace(path.Get.Tags[0]))
		}
		if path.HasMethodWithTag(http.MethodPatch) {
			pman = postmanAddItemToFolder(pman,
				Swagger2PathToPostman2APIItem(cfg, swag, u, http.MethodPatch, path.Patch),
				strings.TrimSpace(path.Patch.Tags[0]))
		}
		if path.HasMethodWithTag(http.MethodPost) {
			pman = postmanAddItemToFolder(pman,
				Swagger2PathToPostman2APIItem(cfg, swag, u, http.MethodPost, path.Post),
				strings.TrimSpace(path.Post.Tags[0]))
		}
		if path.HasMethodWithTag(http.MethodPut) {
			pman = postmanAddItemToFolder(pman,
				Swagger2PathToPostman2APIItem(cfg, swag, u, http.MethodPut, path.Put),
				strings.TrimSpace(path.Put.Tags[0]))
		}
		if path.HasMethodWithTag(http.MethodDelete) {
			pman = postmanAddItemToFolder(pman,
				Swagger2PathToPostman2APIItem(cfg, swag, u, http.MethodDelete, path.Delete),
				strings.TrimSpace(path.Delete.Tags[0]))
		}
	}

	return pman
}

func postmanAddItemToFolder(pman postman2.Collection, pmItem *postman2.Item, pmFolderName string) postman2.Collection {
	if pmItem != nil {
		pmFolder := pman.GetOrNewFolder(pmFolderName)
		pmFolder.Item = append(pmFolder.Item, pmItem)
		pman.SetFolder(pmFolder)
	}
	return pman
}

// Swagger2PathToPostman2APIItem converts a Swagger 2.0 path to a
// Postman 2.0 API item
func Swagger2PathToPostman2APIItem(cfg Configuration, swag openapi2.Specification, url string, method string, endpoint *openapi2.Endpoint) *postman2.Item {
	pmURL := BuildPostmanURL(cfg, swag, url, endpoint)
	item := &postman2.Item{
		Name: endpoint.Summary,
		Request: &postman2.Request{
			Method: strings.ToUpper(method),
			URL:    &pmURL,
		},
	}

	headers := cfg.PostmanHeaders

	headers, requestContentType := postman2.AppendPostmanHeaderValueLower(
		headers,
		httputilmore.HeaderContentType,
		endpoint.Consumes,
		postman2.DefaultMediaTypePreferencesSlice())

	headers, _ = postman2.AppendPostmanHeaderValueLower(
		headers,
		httputilmore.HeaderAccept,
		endpoint.Produces,
		postman2.DefaultMediaTypePreferencesSlice())

	item.Request.Header = headers

	jsonCT := httputilmore.ContentTypeAppJSONUtf8
	indexAppJSON := strings.Index(
		strings.ToLower(requestContentType), jsonCT)
	if indexAppJSON > -1 {
		jsonExample, err := openapi2.GetJSONBodyParameterExampleForKey(
			endpoint.Parameters, jsonCT)
		if err == nil {
			jsonExample = strings.TrimSpace(jsonExample)
			if len(jsonExample) >= 0 {
				item.Request.Body.Mode = "raw"
				item.Request.Body.Raw = jsonExample
			}
		}
	}
	return item
}

// BuildPostmanURL creates a Postman 2.0 spec URL from a Swagger URL
func BuildPostmanURL(cfg Configuration, swag openapi2.Specification, swaggerURL string, endpoint *openapi2.Endpoint) postman2.URL {
	goURL := url.URL{}

	//URLParts := []string{}
	URLPathParts := []string{}

	cfg.PostmanURLBase = strings.TrimSpace(cfg.PostmanURLBase)
	// Set URL path parts
	if len(cfg.PostmanURLBase) > 0 {
		//URLParts = append(URLParts, cfg.PostmanURLBase)
		goURL.Host = cfg.PostmanURLBase
	} else if len(strings.TrimSpace(cfg.PostmanURLHostname)) > 0 {
		//URLParts = append(URLParts, strings.TrimSpace(cfg.PostmanURLHostname))
		goURL.Host = cfg.PostmanURLHostname
	} else if len(strings.TrimSpace(swag.Host)) > 0 {
		//URLParts = append(URLParts, strings.TrimSpace(swag.Host))
		goURL.Host = swag.Host
	}

	if len(strings.TrimSpace(swag.BasePath)) > 0 {
		//URLParts = append(URLParts, strings.TrimSpace(swag.BasePath))
		URLPathParts = append(URLPathParts, strings.TrimSpace(swag.BasePath))
	}
	if len(strings.TrimSpace(swaggerURL)) > 0 {
		//URLParts = append(URLParts, strings.TrimSpace(swaggerURL))
		URLPathParts = append(URLPathParts, strings.TrimSpace(swaggerURL))
	}

	// Create URL
	rx1 := regexp.MustCompile(`/+`)
	rx2 := regexp.MustCompile(`^/+`)
	/*
		rawPostmanURL := strings.TrimSpace(strings.Join(URLParts, "/"))
		rawPostmanURL = rx1.ReplaceAllString(rawPostmanURL, "/")
		rawPostmanURL = rx2.ReplaceAllString(rawPostmanURL, "")
	*/

	rawPostmanURLPath := strings.TrimSpace(strings.Join(URLPathParts, "/"))
	rawPostmanURLPath = rx1.ReplaceAllString(rawPostmanURLPath, "/")
	rawPostmanURLPath = rx2.ReplaceAllString(rawPostmanURLPath, "")
	goURL.Path = rawPostmanURLPath

	// Add URL Scheme
	if len(cfg.PostmanURLBase) == 0 {
		if len(swag.Schemes) > 0 {
			for _, scheme := range swag.Schemes {
				if len(strings.TrimSpace(scheme)) > 0 {
					//rawPostmanURL = strings.Join([]string{scheme, rawPostmanURL}, "://")
					goURL.Scheme = scheme
					break
				}
			}
		}
	}

	/*
		rx3 := regexp.MustCompile(`(^|[^\{])\{([^\/\{\}]+)\}([^\}]|$)`)
		rawPostmanURL = rx3.ReplaceAllString(rawPostmanURL, "$1:$2$3")
		goUrl.Path = rx3.ReplaceAllString(goUrl.Path, "$1:$2$3")
	*/

	goURL.Path = postman2.APIURLOasToPostman(goURL.Path)
	postmanURL := postman2.NewURLForGoURL(goURL)

	// rawPostmanURL = postman2.ApiUrlOasToPostman(rawPostmanURL)
	// postmanURL := postman2.NewURL(rawPostmanURL)

	// Set Default URL Path Parameters
	rx4 := regexp.MustCompile(`^\s*(:(.+))\s*$`)

	for _, part := range postmanURL.Path {
		rs4 := rx4.FindAllStringSubmatch(part, -1)
		if len(rs4) > 0 {
			baseVariable := rs4[0][2]
			var defaultValue interface{}
			for _, parameter := range endpoint.Parameters {
				if parameter.Name == baseVariable {
					defaultValue = parameter.Default
					break
				}
			}
			postmanURL.AddVariable(baseVariable, defaultValue)
		}
	}

	return postmanURL
}
