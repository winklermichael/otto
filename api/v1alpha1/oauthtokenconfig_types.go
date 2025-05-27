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

// OAuthTokenConfigSpec defines the desired state of OAuthTokenConfig
type OAuthTokenConfigSpec struct {
	// URL to refresh the token
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern=`^https?://[a-zA-Z0-9_.-]+(:[0-9]+)?(/.*)?$`
	// +kubebuilder:validation:MaxLength=2048
	// +kubebuilder:validation:MinLength=1
	RefreshURL string `json:"refreshUrl"`

	// Reference to the secret containing client credentials
	// +kubebuilder:validation:Required
	CredentialsSecretRef corev1.SecretReference `json:"credentialsSecretRef"`

	// Reference to the secret where the token will be written
	// +kubebuilder:validation:Required
	TargetSecretRef corev1.SecretReference `json:"targetSecretRef"`

	// OAuth Grant type, one of "ropc", "todo"
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=ropc;todo
	Type string `json:"type"`

	// Optional: the name of the field in the target secret where the token will be stored
	// +kubebuilder:validation:Pattern=^[a-zA-Z0-9_.-]+$
	// +kubebuilder:validation:MaxLength=64
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:default="access_token"
	TokenFieldName string `json:"tokenFieldName,omitempty"`

	// Optional: the name of the field in the target secret where the refresh token will be stored
	// +kubebuilder:validation:Pattern=^[a-zA-Z0-9_.-]+$
	// +kubebuilder:validation:MaxLength=64
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:default="refresh_token"
	RefreshTokenFieldName string `json:"refreshTokenFieldName,omitempty"`

	// Optional: the name of the field in the credentials secret where the client ID is stored
	// Default: "client_id"
	// +kubebuilder:validation:Pattern=^[a-zA-Z0-9_.-]+$
	// +kubebuilder:validation:MaxLength=64
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:default="client_id"
	ClientIDFieldName string `json:"clientIdFieldName,omitempty"`

	// Optional: the name of the field in the credentials secret where the client secret is stored
	// Default: "client_secret"
	// +kubebuilder:validation:Pattern=^[a-zA-Z0-9_.-]+$
	// +kubebuilder:validation:MaxLength=64
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:default="client_secret"
	ClientSecretFieldName string `json:"clientSecretFieldName,omitempty"`

	// Optional: the name of the field in the credentials secret where the username is stored
	// Default: "username"
	// +kubebuilder:validation:Pattern=^[a-zA-Z0-9_.-]+$
	// +kubebuilder:validation:MaxLength=64
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:default="username"
	UsernameFieldName string `json:"usernameFieldName,omitempty"`

	// Optional: the name of the field in the credentials secret where the password is stored
	// Default: "password"
	// +kubebuilder:validation:Pattern=^[a-zA-Z0-9_.-]+$
	// +kubebuilder:validation:MaxLength=64
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:default="password"
	PasswordFieldName string `json:"passwordFieldName,omitempty"`

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
	LastRefreshed     metav1.Time `json:"lastRefreshed,omitempty"`
	NextRefresh       metav1.Time `json:"nextRefresh,omitempty"`
	ExpirationTime    metav1.Time `json:"expirationTime,omitempty"`
	RefreshExpiration metav1.Time `json:"refreshExpiration,omitempty"`
	Status            string      `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Last Refreshed",type=string,JSONPath=`.status.lastRefreshed`,description="The last time the token was refreshed"
// +kubebuilder:printcolumn:name="Next Refresh",type=string,JSONPath=`.status.nextRefresh`,description="The next scheduled refresh time"
// +kubebuilder:printcolumn:name="Expiration Time",type=string,JSONPath=`.status.expirationTime`,description="The token expiration time"
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
