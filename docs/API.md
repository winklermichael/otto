# OAuthTokenConfig API

The `OAuthTokenConfig` Custom Resource Definition (CRD) is used to manage OAuth token configurations in Kubernetes. Below is the detailed specification of the CRD.

## Specification

### Metadata
- **apiVersion**: `auth.example.com/v1alpha1`
- **kind**: `OAuthTokenConfig`
- **metadata**: Standard Kubernetes object metadata.

### Spec Fields

| Field                     | Type                | Description                                                                                          | Required | Default Value       |
|---------------------------|---------------------|------------------------------------------------------------------------------------------------------|----------|---------------------|
| `tokenUrl`                | `string`           | URL to refresh the token. Must be a valid HTTP/HTTPS URL.                                           | Yes      | N/A                 |
| `type`                    | `string`           | OAuth Grant type. Must be one of `["ropc"]`.                                                        | Yes      | N/A                 |
| `target`                  | `TargetConfig`     | Configuration for the target secret where the token will be written.                                | Yes      | N/A                 |
| `credentials`             | `CredentialsConfig`| Configuration for the credentials secret containing client credentials.                             | Yes      | N/A                 |
| `tokenResponse`           | `TokenResponseConfig` | Configuration for the token response fields.                                                        | No       | See defaults below. |
| `tokenRequest`            | `TokenRequestConfig` | Configuration for the token request fields.                                                         | No       | See defaults below. |
| `refreshInterval`         | `Duration`         | Time interval between token refreshes.                                                              | No       | N/A                 |
| `refreshBufferPercentage` | `int32`            | Percentage of token expiration time before refresh. Must be between 0 and 100.                      | No       | `10`                |

#### TargetConfig Fields

| Field                     | Type                | Description                                                                                          | Required | Default Value       |
|---------------------------|---------------------|------------------------------------------------------------------------------------------------------|----------|---------------------|
| `secretRef`               | `SecretReference`  | Reference to the secret where the token will be written.                                            | Yes      | N/A                 |
| `accessTokenFieldName`    | `string`           | Name of the field in the target secret where the token will be stored.                              | No       | `access_token`      |
| `refreshTokenFieldName`   | `string`           | Name of the field in the target secret where the refresh token will be stored.                      | No       | `refresh_token`     |

#### CredentialsConfig Fields

| Field                     | Type                | Description                                                                                          | Required | Default Value       |
|---------------------------|---------------------|------------------------------------------------------------------------------------------------------|----------|---------------------|
| `secretRef`               | `SecretReference`  | Reference to the secret containing client credentials.                                              | Yes      | N/A                 |
| `clientIdFieldName`       | `string`           | Name of the field in the credentials secret where the client ID is stored.                          | No       | `client_id`         |
| `clientSecretFieldName`   | `string`           | Name of the field in the credentials secret where the client secret is stored.                      | No       | `client_secret`     |
| `usernameFieldName`       | `string`           | Name of the field in the credentials secret where the username is stored.                           | No       | `username`          |
| `passwordFieldName`       | `string`           | Name of the field in the credentials secret where the password is stored.                           | No       | `password`          |

#### TokenResponseConfig Fields

| Field                     | Type                | Description                                                                                          | Required | Default Value       |
|---------------------------|---------------------|------------------------------------------------------------------------------------------------------|----------|---------------------|
| `accessTokenFieldName`    | `string`           | Name of the field in the token response where the access token is stored.                           | No       | `access_token`      |
| `refreshTokenFieldName`   | `string`           | Name of the field in the token response where the refresh token is stored.                          | No       | `refresh_token`     |
| `expirationFieldName`     | `string`           | Name of the field in the token response where the expiration time is stored.                        | No       | `expires_in`        |
| `refreshExpirationFieldName` | `string`        | Name of the field in the token response where the refresh expiration time is stored.                | No       | `refresh_expires_in`|

#### TokenRequestConfig Fields

| Field                     | Type                | Description                                                                                          | Required | Default Value       |
|---------------------------|---------------------|------------------------------------------------------------------------------------------------------|----------|---------------------|
| `method`                  | `string`           | HTTP method to use for the token request. Must be one of `["POST", "GET"]`.                         | No       | `POST`              |
| `contentType`             | `string`           | Content type of the token request. Must be one of `["application/x-www-form-urlencoded", "application/json"]`. | No | `application/x-www-form-urlencoded` |
| `headers`                 | `map[string]string`| Additional headers to include in the token request.                                                 | No       | N/A                 |
| `grantTypeFieldName`      | `string`           | Name of the field for the grant type in the token request.                                          | No       | `grant_type`        |
| `clientIdFieldName`       | `string`           | Name of the field for the client ID in the token request.                                           | No       | `client_id`         |
| `clientSecretFieldName`   | `string`           | Name of the field for the client secret in the token request.                                       | No       | `client_secret`     |
| `usernameFieldName`       | `string`           | Name of the field for the username in the token request.                                            | No       | `username`          |
| `passwordFieldName`       | `string`           | Name of the field for the password in the token request.                                            | No       | `password`          |
| `refreshTokenFieldName`   | `string`           | Name of the field for the refresh token in the token request.                                       | No       | `refresh_token`     |

### Status Fields

| Field                     | Type       | Description                                                                                          |
|---------------------------|------------|------------------------------------------------------------------------------------------------------|
| `lastRefresh`             | `Time`     | The last time the token was refreshed.                                                              |
| `nextRefresh`             | `Time`     | The next scheduled refresh time.                                                                    |
| `expirationTime`          | `Time`     | The token expiration time.                                                                          |
| `refreshExpirationTime`   | `Time`     | The refresh token expiration time.                                                                  |
| `status`                  | `string`   | The current status of the resource.                                                                 |