package configuration

import (
	"gopkg.in/yaml.v3"
)

type Flow struct {
	Name          FlowName        `yaml:"name"`
	Requirement   FlowRequirement `yaml:"requirement,omitempty"`
	Authorization Authorization   `yaml:"authorization"`
	Environment   Environment     `yaml:"environment"`
}

func (this *Flow) SetDefaults() error {
	return setDefaults(this,
		noopSetDefault[Flow]("name"),
		func(v *Flow) (string, defaulter) { return "requirement", &v.Requirement },
		func(v *Flow) (string, defaulter) { return "authorization", &v.Authorization },
		func(v *Flow) (string, defaulter) { return "environment", &v.Environment },
	)
}

func (this *Flow) Trim() error {
	return trim(this,
		noopTrim[Flow]("name"),
		func(v *Flow) (string, trimmer) { return "requirement", &v.Requirement },
		func(v *Flow) (string, trimmer) { return "authorization", &v.Authorization },
		func(v *Flow) (string, trimmer) { return "environment", &v.Environment },
	)
}

func (this *Flow) Validate() error {
	return validate(this,
		notZeroValidate("name", func(v *Flow) *FlowName { return &v.Name }),
		func(v *Flow) (string, validator) { return "requirement", &v.Requirement },
		func(v *Flow) (string, validator) { return "authorization", &v.Authorization },
		func(v *Flow) (string, validator) { return "environment", &v.Environment },
	)
}

func (this *Flow) UnmarshalYAML(node *yaml.Node) error {
	return unmarshalYAML(this, node, func(target *Flow, node *yaml.Node) error {
		type raw Flow
		return node.Decode((*raw)(target))
	})
}

func (this Flow) IsEqualTo(other any) bool {
	if other == nil {
		return false
	}
	switch v := other.(type) {
	case Flow:
		return this.isEqualTo(&v)
	case *Flow:
		return this.isEqualTo(v)
	default:
		return false
	}
}

func (this Flow) isEqualTo(other *Flow) bool {
	return isEqual(&this.Name, &other.Name) &&
		isEqual(&this.Requirement, &other.Requirement) &&
		isEqual(&this.Authorization, &other.Authorization) &&
		isEqual(&this.Environment, &other.Environment)
}

type Flows []Flow

func (this *Flows) SetDefaults() error {
	return setSliceDefaults(this)
}

func (this *Flows) Trim() error {
	return trimSlice(this)
}

func (this Flows) Validate() error {
	return validateSlice(this)
}

func (this Flows) IsEqualTo(other any) bool {
	if other == nil {
		return false
	}
	switch v := other.(type) {
	case Flows:
		return this.isEqualTo(&v)
	case *Flows:
		return this.isEqualTo(v)
	default:
		return false
	}
}

func (this Flows) isEqualTo(other *Flows) bool {
	if len(this) != len(*other) {
		return false
	}
	for i, tv := range this {
		if !tv.IsEqualTo((*other)[i]) {
			return false
		}
	}
	return true
}