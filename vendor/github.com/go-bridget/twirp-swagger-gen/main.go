package main

import (
	"flag"
	"fmt"
	"os"
	"path"
	"strings"

	"encoding/json"
	"io/ioutil"

	"github.com/apex/log"
	"github.com/davecgh/go-spew/spew"
	"github.com/emicklei/proto"
	"github.com/go-openapi/spec"
	"github.com/pkg/errors"
)

var _ = spew.Dump

type SwaggerWriter struct {
	*spec.Swagger

	hostname    string
	filename    string
	packageName string
}

func NewSwaggerWriter(filename string, hostname string) *SwaggerWriter {
	return &SwaggerWriter{
		filename: filename,
		hostname: hostname,
		Swagger:  &spec.Swagger{},
	}
}

func (sw *SwaggerWriter) Package(pkg *proto.Package) {
	sw.Swagger.Swagger = "2.0"
	sw.Schemes = []string{"http", "https"}
	sw.Produces = []string{"application/json"}
	sw.Host = sw.hostname
	sw.Consumes = sw.Produces
	sw.Info = &spec.Info{
		InfoProps: spec.InfoProps{
			Title:   path.Base(sw.filename),
			Version: "version not set",
		},
	}
	sw.Swagger.Definitions = make(spec.Definitions)
	sw.Swagger.Paths = &spec.Paths{
		Paths: make(map[string]spec.PathItem),
	}

	sw.packageName = pkg.Name
}

func (sw *SwaggerWriter) Import(i *proto.Import) {
	// the exclusion here is more about path traversal than it is
	// about the structure of google proto messages. The annotations
	// could serve to document a REST API, which goes beyond what
	// Twitch RPC does out of the box.
	if strings.Contains(i.Filename, "google/api/annotations.proto") {
		return
	}

	log.Debugf("importing %s", i.Filename)

	definition, err := loadProtoFile(i.Filename)
	if err != nil {
		log.Infof("Can't load %s, err=%s, ignoring", i.Filename, err)
		return
	}

	oldPackageName := sw.packageName

	withPackage := func(pkg *proto.Package) {
		sw.packageName = pkg.Name
	}

	// additional files walked for messages and imports only
	proto.Walk(definition, proto.WithPackage(withPackage), proto.WithImport(sw.Import), proto.WithMessage(sw.Message))

	sw.packageName = oldPackageName
}

func comment(comment *proto.Comment) string {
	if comment == nil {
		return ""
	}

	result := ""
	for _, line := range comment.Lines {
		line = strings.TrimSpace(line)
		if line == "" {
			break
		}
		result += " " + line
	}
	if len(result) > 1 {
		return result[1:]
	}
	return ""
}

func description(comment *proto.Comment) string {
	if comment == nil {
		return ""
	}

	grab := false

	result := []string{}
	for _, line := range comment.Lines {
		line = strings.TrimSpace(line)
		if line == "" {
			if grab {
				break
			}
			grab = true
			continue
		}
		if grab {
			result = append(result, line)
		}
	}
	return strings.Join(result, "\n")
}

func (sw *SwaggerWriter) RPC(rpc *proto.RPC) {
	parent, ok := rpc.Parent.(*proto.Service)
	if !ok {
		panic("parent is not proto.service")
	}

	pathName := fmt.Sprintf("/twirp/%s.%s/%s", sw.packageName, parent.Name, rpc.Name)

	sw.Swagger.Paths.Paths[pathName] = spec.PathItem{
		PathItemProps: spec.PathItemProps{
			Post: &spec.Operation{
				OperationProps: spec.OperationProps{
					ID:      rpc.Name,
					Tags:    []string{parent.Name},
					Summary: comment(rpc.Comment),
					Responses: &spec.Responses{
						ResponsesProps: spec.ResponsesProps{
							StatusCodeResponses: map[int]spec.Response{
								200: spec.Response{
									ResponseProps: spec.ResponseProps{
										Description: "A successful response.",
										Schema: &spec.Schema{
											SchemaProps: spec.SchemaProps{
												Ref: spec.MustCreateRef(fmt.Sprintf("#/definitions/%s%s", sw.packageName, rpc.ReturnsType)),
											},
										},
									},
								},
							},
						},
					},
					Parameters: []spec.Parameter{
						spec.Parameter{
							ParamProps: spec.ParamProps{
								Name:     "body",
								In:       "body",
								Required: true,
								Schema: &spec.Schema{
									SchemaProps: spec.SchemaProps{
										Ref: spec.MustCreateRef(fmt.Sprintf("#/definitions/%s%s", sw.packageName, rpc.RequestType)),
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func (sw *SwaggerWriter) Message(msg *proto.Message) {
	definitionName := fmt.Sprintf("%s%s", sw.packageName, msg.Name)

	schemaProps := make(map[string]spec.Schema)

	var allowedValues = []string{
		"boolean",
		"integer",
		"number",
		"object",
		"string",
	}

	find := func(haystack []string, needle string) (int, bool) {
		for k, v := range haystack {
			if v == needle {
				return k, true
			}
		}
		return -1, false
	}

	var fieldOrder = []string{}

	for _, element := range msg.Elements {
		switch val := element.(type) {
		case *proto.Oneof:
			// TODO: feel free to submit a PR. The fields listed in "oneof"
			// should be unpacked and handled like *proto.NormalField below
			log.Infof("Need to implement *proto.Oneof, want to make a PR?")
			break
		case *proto.NormalField:
			var (
				fieldTitle       = comment(val.Field.Comment)
				fieldDescription = description(val.Field.Comment)
				fieldName        = val.Field.Name
				fieldType        = val.Field.Type
				fieldFormat      = val.Field.Type
			)
			if fieldType == "bool" {
				fieldType = "boolean"
				fieldFormat = "boolean"
			}
			if fieldType == "int64" || fieldType == "uint64" {
				fieldType = "string"
			}
			if strings.HasPrefix(fieldType, "int") || strings.HasPrefix(fieldType, "uint") {
				fieldType = "integer"
			}
			if fieldType != "boolean" && fieldType == fieldFormat {
				fieldFormat = ""
			}

			fieldOrder = append(fieldOrder, fieldName)

			if _, ok := find(allowedValues, fieldType); ok {
				fieldSchema := spec.Schema{
					SchemaProps: spec.SchemaProps{
						Title:       fieldTitle,
						Description: fieldDescription,
						Type:        spec.StringOrArray([]string{fieldType}),
						Format:      fieldFormat,
					},
				}
				if val.Repeated {
					fieldSchema.Title = ""
					fieldSchema.Description = ""
					fieldSchema.Format = ""
					schemaProps[fieldName] = spec.Schema{
						SchemaProps: spec.SchemaProps{
							Title:       fieldTitle,
							Description: fieldDescription,
							Type:        spec.StringOrArray([]string{"array"}),
							Format:      fieldFormat,
							Items: &spec.SchemaOrArray{
								Schema: &fieldSchema,
							},
						},
					}
				} else {
					schemaProps[fieldName] = fieldSchema
				}
				continue
			}

			// Prefix rich type with package name
			if !strings.Contains(fieldType, ".") {
				fieldType = sw.packageName + "." + fieldType
			}
			ref := fmt.Sprintf("#/definitions/%s", strings.ReplaceAll(fieldType, ".", ""))
			// fmt.Sprintf("#/definitions/%s%s", sw.packageName, fieldType)

			if val.Repeated {
				schemaProps[fieldName] = spec.Schema{
					SchemaProps: spec.SchemaProps{
						Title:       fieldTitle,
						Description: fieldDescription,
						Type:        spec.StringOrArray([]string{"array"}),
						Items: &spec.SchemaOrArray{
							Schema: &spec.Schema{
								SchemaProps: spec.SchemaProps{
									Ref: spec.MustCreateRef(ref),
								},
							},
						},
					},
				}
				continue
			}
			schemaProps[fieldName] = spec.Schema{
				SchemaProps: spec.SchemaProps{
					Title:       fieldTitle,
					Description: fieldDescription,
					Ref:         spec.MustCreateRef(ref),
				},
			}
		default:
			log.Infof("Unknown field type: %T", element)
		}
	}

	schemaDesc := description(msg.Comment)
	if len(fieldOrder) > 0 {
		schemaDesc = schemaDesc + "\n\nFields: " + strings.Join(fieldOrder, ", ")
	}

	sw.Swagger.Definitions[definitionName] = spec.Schema{
		SchemaProps: spec.SchemaProps{
			Title:       comment(msg.Comment),
			Description: strings.TrimSpace(schemaDesc),
			Type:        spec.StringOrArray([]string{"object"}),
			Properties:  schemaProps,
		},
	}
}

func (sw *SwaggerWriter) Handlers() []proto.Handler {
	return []proto.Handler{
		proto.WithPackage(sw.Package),
		proto.WithRPC(sw.RPC),
		proto.WithMessage(sw.Message),
		proto.WithImport(sw.Import),
	}
}

func (sw *SwaggerWriter) Save(filename string) error {
	body := sw.Get()
	return ioutil.WriteFile(filename, body, os.ModePerm^0111)
}
func (sw *SwaggerWriter) Get() []byte {
	b, _ := json.MarshalIndent(sw, "", "  ")
	return b
}

func loadProtoFile(filename string) (*proto.Proto, error) {
	reader, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	parser := proto.NewParser(reader)
	return parser.Parse()
}

func parse(hostname, filename, output string) error {
	if filename == output {
		return errors.New("output file must be different than input file")
	}

	writer := NewSwaggerWriter(filename, hostname)

	definition, err := loadProtoFile(filename)
	if err != nil {
		return err
	}

	// main file for all the relevant info
	proto.Walk(definition, writer.Handlers()...)

	return writer.Save(output)
}

func main() {
	var (
		in   string
		out  string
		host string
	)
	flag.StringVar(&in, "in", "", "Input source .proto file")
	flag.StringVar(&out, "out", "", "Output swagger.json file")
	flag.StringVar(&host, "host", "api.example.com", "API host name")
	flag.Parse()

	if in == "" {
		log.Fatalf("Missing parameter: -in [input.proto]")
	}
	if out == "" {
		log.Fatalf("Missing parameter: -out [output.proto]")
	}
	if host == "" {
		log.Fatalf("Missing parameter: -host [api.example.com]")
	}

	if err := parse(host, in, out); err != nil {
		log.WithError(err).Fatal("exit with error")
	}
}
