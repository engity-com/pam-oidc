package configuration

import (
	"github.com/engity-com/yasshd/pkg/common"
	"github.com/engity-com/yasshd/pkg/crypto"
	"github.com/engity-com/yasshd/pkg/template"
	"testing"
)

func TestConfiguration_UnmarshalYAML(t *testing.T) {
	runUnmarshalYamlTests(t,
		unmarshalYamlTestCase[Configuration]{
			name:          "empty",
			yaml:          ``,
			expectedError: `EOF`,
		},
		unmarshalYamlTestCase[Configuration]{
			name:          "flows-missing",
			yaml:          `{}`,
			expectedError: `[flows] required but absent`,
		},
		unmarshalYamlTestCase[Configuration]{
			name:          "flows-empty",
			yaml:          `flows: []`,
			expectedError: `[flows] required but absent`,
		},
		unmarshalYamlTestCase[Configuration]{
			name: "required-set",
			yaml: `flows:
- name: foo
  authorization: 
    type: oidc
    issuer: https://foo-bar
    clientId: anId
    clientSecret: aSecret`,
			expected: Configuration{
				Ssh: Ssh{
					Addresses: DefaultSshAddresses,
					Keys: Keys{
						HostKeys:           DefaultHostKeyLocations,
						RsaRestriction:     crypto.DefaultRsaRestriction,
						DsaRestriction:     crypto.DefaultDsaRestriction,
						EcdsaRestriction:   crypto.DefaultEcdsaRestriction,
						Ed25519Restriction: crypto.DefaultEd25519Restriction,
					},
				},
				Flows: []Flow{{
					Name: "foo",
					Requirement: FlowRequirement{
						IncludedRequestingName: common.MustNewRegexp(""),
						ExcludedRequestingName: common.MustNewRegexp(""),
					},
					Authorization: Authorization{&AuthorizationOidcDeviceAuth{
						Issuer:           "https://foo-bar",
						ClientId:         "anId",
						ClientSecret:     "aSecret",
						Scopes:           DefaultAuthorizationOidcScopes,
						RetrieveIdToken:  true,
						RetrieveUserInfo: false,
					}},
					Environment: Environment{&EnvironmentLocal{
						User: UserRequirementTemplate{
							Name:        template.MustNewString(DefaultUserNameTmpl),
							DisplayName: template.MustNewString(DefaultUserDisplayNameTmpl),
							Group: GroupRequirementTemplate{
								Name: template.MustNewString(DefaultGroupNameTmpl),
							},
							Shell:   template.MustNewString(DefaultUserShellTmpl),
							HomeDir: template.MustNewString(DefaultUserHomeDirTmpl),
						},
						LoginAllowed:      template.MustNewBool(DefaultEnvironmentLocalLoginAllowedTmpl),
						CreateIfAbsent:    template.MustNewBool(DefaultEnvironmentLocalCreateIfAbsentTmpl),
						UpdateIfDifferent: template.MustNewBool(DefaultEnvironmentLocalUpdateIfDifferentTmpl),
					}},
				}},
			},
		},
	)
}