package configuration

import (
	"github.com/coreos/go-oidc/v3/oidc"
	"gopkg.in/yaml.v3"
	"slices"
)

var (
	DefaultAuthorizationOidcScopes = []string{oidc.ScopeOpenID, "profile", "email"}
)

type AuthorizationOidcDeviceAuth struct {
	Issuer       string   `yaml:"issuer"`
	ClientId     string   `yaml:"clientId"`
	ClientSecret string   `yaml:"clientSecret"`
	Scopes       []string `yaml:"scopes"`

	RetrieveIdToken  bool `yaml:"retrieveIdToken,omitempty"`
	RetrieveUserInfo bool `yaml:"retrieveUserInfo,omitempty"`
}

func (this *AuthorizationOidcDeviceAuth) SetDefaults() error {
	return setDefaults(this,
		noopSetDefault[AuthorizationOidcDeviceAuth]("issuer"),
		noopSetDefault[AuthorizationOidcDeviceAuth]("clientId"),
		noopSetDefault[AuthorizationOidcDeviceAuth]("clientSecret"),
		fixedDefault("scopes", func(v *AuthorizationOidcDeviceAuth) *[]string { return &v.Scopes }, DefaultAuthorizationOidcScopes),

		fixedDefault("retrieveIdToken", func(v *AuthorizationOidcDeviceAuth) *bool { return &v.RetrieveIdToken }, true),
		fixedDefault("retrieveUserInfo", func(v *AuthorizationOidcDeviceAuth) *bool { return &v.RetrieveUserInfo }, false),
	)
}

func (this *AuthorizationOidcDeviceAuth) Trim() error {
	return trim(this,
		noopTrim[AuthorizationOidcDeviceAuth]("issuer"),
		noopTrim[AuthorizationOidcDeviceAuth]("clientId"),
		noopTrim[AuthorizationOidcDeviceAuth]("clientSecret"),
		trimSliceBy("scopes", func(v *AuthorizationOidcDeviceAuth) *[]string { return &v.Scopes }, func(v string) bool { return v == "" }),

		noopTrim[AuthorizationOidcDeviceAuth]("retrieveIdToken"),
		noopTrim[AuthorizationOidcDeviceAuth]("retrieveUserInfo"),
	)
}

func (this *AuthorizationOidcDeviceAuth) Validate() error {
	return validate(this,
		notEmptyStringValidate("issuer", func(v *AuthorizationOidcDeviceAuth) *string { return &v.Issuer }),
		notEmptyStringValidate("clientId", func(v *AuthorizationOidcDeviceAuth) *string { return &v.ClientId }),
		notEmptyStringValidate("clientSecret", func(v *AuthorizationOidcDeviceAuth) *string { return &v.ClientSecret }),
		notEmptySliceValidate("scopes", func(v *AuthorizationOidcDeviceAuth) *[]string { return &v.Scopes }),

		noopValidate[AuthorizationOidcDeviceAuth]("retrieveIdToken"),
		noopValidate[AuthorizationOidcDeviceAuth]("retrieveUserInfo"),
	)
}

func (this *AuthorizationOidcDeviceAuth) UnmarshalYAML(node *yaml.Node) error {
	return unmarshalYAML(this, node, func(target *AuthorizationOidcDeviceAuth, node *yaml.Node) error {
		type raw AuthorizationOidcDeviceAuth
		return node.Decode((*raw)(target))
	})
}

func (this AuthorizationOidcDeviceAuth) IsEqualTo(other any) bool {
	if other == nil {
		return false
	}
	switch v := other.(type) {
	case AuthorizationOidcDeviceAuth:
		return this.isEqualTo(&v)
	case *AuthorizationOidcDeviceAuth:
		return this.isEqualTo(v)
	default:
		return false
	}
}

func (this AuthorizationOidcDeviceAuth) isEqualTo(other *AuthorizationOidcDeviceAuth) bool {
	return this.Issuer == other.Issuer &&
		this.ClientId == other.ClientId &&
		slices.EqualFunc(this.Scopes, other.Scopes, func(l, r string) bool { return l == r }) &&
		this.RetrieveIdToken == other.RetrieveIdToken &&
		this.RetrieveUserInfo == other.RetrieveUserInfo
}