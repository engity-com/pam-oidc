package configuration

import (
	"gopkg.in/yaml.v3"

	"github.com/engity-com/bifroest/pkg/template"
)

var (
	DefaultImpAlternativesDownloadUrl = template.MustNewUrl("https://github.com/engity-com/bifroest/releases/download/v{{.Version}}/bifroest-{{.Os}}-{{.Architecture}}-generic.{{if eq .Os `windows`}}zip{{else}}tgz{{end}}")
	DefaultImpAlternativesLocation    = template.MustNewString(defaultImpAlternativesLocation)
)

type Imp struct {
	AlternativesDownloadUrl template.Url    `yaml:"alternativesDownloadUrl,omitempty"`
	AlternativesLocation    template.String `yaml:"alternativesLocation,omitempty"`
}

func (this *Imp) SetDefaults() error {
	return setDefaults(this,
		fixedDefault("alternativesDownloadUrl", func(v *Imp) *template.Url { return &v.AlternativesDownloadUrl }, DefaultImpAlternativesDownloadUrl),
		fixedDefault("alternativesLocation", func(v *Imp) *template.String { return &v.AlternativesLocation }, DefaultImpAlternativesLocation),
	)
}

func (this *Imp) Trim() error {
	return trim(this,
		noopTrim[Imp]("alternativesDownloadUrl"),
		noopTrim[Imp]("alternativesLocation"),
	)
}

func (this *Imp) Validate() error {
	return validate(this,
		noopValidate[Imp]("alternativesDownloadUrl"),
		noopValidate[Imp]("alternativesLocation"),
	)
}

func (this *Imp) UnmarshalYAML(node *yaml.Node) error {
	return unmarshalYAML(this, node, func(target *Imp, node *yaml.Node) error {
		type raw Imp
		return node.Decode((*raw)(target))
	})
}

func (this Imp) IsEqualTo(other any) bool {
	if other == nil {
		return false
	}
	switch v := other.(type) {
	case Imp:
		return this.isEqualTo(&v)
	case *Imp:
		return this.isEqualTo(v)
	default:
		return false
	}
}

func (this Imp) isEqualTo(other *Imp) bool {
	return isEqual(&this.AlternativesDownloadUrl, &other.AlternativesDownloadUrl) &&
		isEqual(&this.AlternativesLocation, &other.AlternativesLocation)
}
