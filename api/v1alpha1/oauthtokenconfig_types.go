/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TargetConfig groups fields related to the target secret
type TargetConfig struct {
	// Reference to the secret where the token will be written
	// +kubebuilder:validation:Required
	SecretRef corev1.SecretReference `json:"secretRef"`

	// Optional: the name of the field in the target secret where the token will be stored
	// +kubebuilder:validation:Pattern=^[a-zA-Z0-9_.-]+$
	// +kubebuilder:validation:MaxLength=64
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:default="access_token"
	AccessTokenFieldName string `json:"accessTokenFieldName,omitempty"`

	// Optional: the name of the field in the target secret where the refresh token will be stored
	// +kubebuilder:validation:Pattern=^[a-zA-Z0-9_.-]+$
	// +kubebuilder:validation:MaxLength=64
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:default="refresh_token"
	RefreshTokenFieldName string `json:"refreshTokenFieldName,omitempty"`
}

// CredentialsConfig groups fields related to the credentials secret
type CredentialsConfig struct {
	// Reference to the secret containing client credentials
	// +kubebuilder:validation:Required
	SecretRef corev1.SecretReference `json:"secretRef"`

	// Optional: the name of the field in the credentials secret where the client ID is stored
	// +kubebuilder:validation:Pattern=^[a-zA-Z0-9_.-]+$
	// +kubebuilder:validation:MaxLength=64
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:default="client_id"
	ClientIDFieldName string `json:"clientIdFieldName,omitempty"`

	// Optional: the name of the field in the credentials secret where the client secret is stored
	// +kubebuilder:validation:Pattern=^[a-zA-Z0-9_.-]+$
	// +kubebuilder:validation:MaxLength=64
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:default="client_secret"
	ClientSecretFieldName string `json:"clientSecretFieldName,omitempty"`

	// Optional: the name of the field in the credentials secret where the username is stored
	// +kubebuilder:validation:Pattern=^[a-zA-Z0-9_.-]+$
	// +kubebuilder:validation:MaxLength=64
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:default="username"
	UsernameFieldName string `json:"usernameFieldName,omitempty"`

	// Optional: the name of the field in the credentials secret where the password is stored
	// +kubebuilder:validation:Pattern=^[a-zA-Z0-9_.-]+$
	// +kubebuilder:validation:MaxLength=64
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:default="password"
	PasswordFieldName string `json:"passwordFieldName,omitempty"`
}

// TokenResponseConfig groups fields related to the token response configuration
type TokenResponseConfig struct {

	// Optional: the name of the field in the token response where the access token is stored
	// +kubebuilder:validation:Pattern=^[a-zA-Z0-9_.-]+$
	// +kubebuilder:validation:MaxLength=64
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:default="access_token"
	AccessTokenFieldName string `json:"accessTokenFieldName,omitempty"`

	// Optional: the name of the field in the token response where the refresh token is stored
	// +kubebuilder:validation:Pattern=^[a-zA-Z0-9_.-]+$
	// +kubebuilder:validation:MaxLength=64
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:default="refresh_token"
	RefreshTokenFieldName string `json:"refreshTokenFieldName,omitempty"`

	// Optional: the name of the field in the token response where the expiration time is stored
	// +kubebuilder:validation:Pattern=^[a-zA-Z0-9_.-]+$
	// +kubebuilder:validation:MaxLength=64
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:default="expires_in"
	ExpirationFieldName string `json:"expirationFieldName,omitempty"`

	// Optional: the name of the field in the token response where the refresh expiration time is stored
	// +kubebuilder:validation:Pattern=^[a-zA-Z0-9_.-]+$
	// +kubebuilder:validation:MaxLength=64
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:default="refresh_expires_in"
	RefreshExpirationFieldName string `json:"refreshExpirationFieldName,omitempty"`
}

// TokenRequestConfig groups fields related to the token request configuration
type TokenRequestConfig struct {
	// Optional: the HTTP method to use for the token request
	// +kubebuilder:validation:Enum=POST;GET
	// +kubebuilder:default=POST
	Method string `json:"method,omitempty"`

	// Optional: the content type of the request
	// +kubebuilder:validation:Enum=application/x-www-form-urlencoded;application/json
	// +kubebuilder:default=application/x-www-form-urlencoded
	ContentType string `json:"contentType,omitempty"`

	// Optional: additional headers to include in the request
	Headers map[string]string `json:"headers,omitempty"`

	// Optional: the field name for the grant type in the token request
	// +kubebuilder:default="grant_type"
	GrantTypeFieldName string `json:"grantTypeFieldName,omitempty"`

	// Optional: the field name for the client ID in the token request
	// +kubebuilder:default="client_id"
	ClientIDFieldName string `json:"clientIdFieldName,omitempty"`

	// Optional: the field name for the client secret in the token request
	// +kubebuilder:default="client_secret"
	ClientSecretFieldName string `json:"clientSecretFieldName,omitempty"`

	// Optional: the field name for the username in the token request
	// +kubebuilder:default="username"
	UsernameFieldName string `json:"usernameFieldName,omitempty"`

	// Optional: the field name for the password in the token request
	// +kubebuilder:default="password"
	PasswordFieldName string `json:"passwordFieldName,omitempty"`

	// Optional: the field name for the refresh token in the token request
	// +kubebuilder:default="refresh_token"
	RefreshTokenFieldName string `json:"refreshTokenFieldName,omitempty"`
}

// OAuthTokenConfigSpec defines the desired state of OAuthTokenConfig
type OAuthTokenConfigSpec struct {
	// URL to refresh the token
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern=`^https?://[a-zA-Z0-9_.-]+(:[0-9]+)?(/.*)?$`
	// +kubebuilder:validation:MaxLength=2048
	// +kubebuilder:validation:MinLength=1
	TokenURL string `json:"tokenUrl"`

	// OAuth Grant type, one of ["ropc"]
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=ropc
	Type string `json:"type"`

	// Configuration for the target secret
	Target TargetConfig `json:"target"`

	// Configuration for the credentials secret
	Credentials CredentialsConfig `json:"credentials"`

	// Configuration for the token response
	// +kubebuilder:default={accessTokenFieldName: "access_token", refreshTokenFieldName: "refresh_token", expirationFieldName: "expires_in", refreshExpirationFieldName: "refresh_expires_in"}
	TokenResponse TokenResponseConfig `json:"tokenResponse"`

	// Configuration for the token request
	// +kubebuilder:default={method: "POST", contentType: "application/x-www-form-urlencoded", grantTypeFieldName: "grant_type", clientIdFieldName: "client_id", clientSecretFieldName: "client_secret", usernameFieldName: "username", passwordFieldName: "password", refreshTokenFieldName: "refresh_token"}
	TokenRequest TokenRequestConfig `json:"tokenRequest"`

	// Optional: time interval between refreshes
	RefreshInterval *metav1.Duration `json:"refreshInterval,omitempty"`

	// Optional: percentage of token expiration time before refresh
	// Default: 10%
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=100
	// +kubebuilder:default=10
	RefreshBufferPercentage int32 `json:"refreshBufferPercentage,omitempty"`
}

// OAuthTokenConfigStatus defines the observed state of OAuthTokenConfig
type OAuthTokenConfigStatus struct {
	LastRefresh           metav1.Time `json:"lastRefresh,omitempty"`
	NextRefresh           metav1.Time `json:"nextRefresh,omitempty"`
	ExpirationTime        metav1.Time `json:"expirationTime,omitempty"`
	RefreshExpirationTime metav1.Time `json:"refreshExpirationTime,omitempty"`
	Status                string      `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Last Refresh",type=string,JSONPath=`.status.lastRefresh`,description="The last time the token was refreshed"
// +kubebuilder:printcolumn:name="Next Refresh",type=string,JSONPath=`.status.nextRefresh`,description="The next scheduled refresh time"
// +kubebuilder:printcolumn:name="Token Expiration Time",type=string,JSONPath=`.status.expirationTime`,description="The token expiration time"
// +kubebuilder:printcolumn:name="Refresh Expiration Time",type=string,JSONPath=`.status.refreshExpirationTime`,description="The refresh token expiration time"
// +kubebuilder:printcolumn:name="Status",type=string,JSONPath=`.status.status`,description="The current status of the resource"

// OAuthTokenConfig is the Schema for the oauthtokenconfigs API
type OAuthTokenConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   OAuthTokenConfigSpec   `json:"spec,omitempty"`
	Status OAuthTokenConfigStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// OAuthTokenConfigList contains a list of OAuthTokenConfig
type OAuthTokenConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []OAuthTokenConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&OAuthTokenConfig{}, &OAuthTokenConfigList{})
}
