/*
 * Minio Cloud Storage, (C) 2018 Minio, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package cmd

import (
	"encoding/xml"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/minio/minio/cmd/logger"
	"github.com/minio/minio/pkg/auth"
	"github.com/minio/minio/pkg/iam/validator"
)

const (
	// STS API version.
	stsAPIVersion = "2011-06-15"
)

// stsAPIHandlers implements and provides http handlers for AWS STS API.
type stsAPIHandlers struct{}

// registerSTSRouter - registers AWS STS compatible APIs.
func registerSTSRouter(router *mux.Router) {
	// Initialize STS.
	sts := &stsAPIHandlers{}

	// STS Router
	stsRouter := router.NewRoute().PathPrefix("/").Subrouter()

	// AssumeRoleWithClientGrants
	stsRouter.Methods("POST").HandlerFunc(httpTraceAll(sts.AssumeRoleWithClientGrants)).
		Queries("Action", "AssumeRoleWithClientGrants")
}

// AssumedRoleUser - The identifiers for the temporary security credentials that
// the operation returns. Please also see https://docs.aws.amazon.com/goto/WebAPI/sts-2011-06-15/AssumedRoleUser
type AssumedRoleUser struct {
	// The ARN of the temporary security credentials that are returned from the
	// AssumeRole action. For more information about ARNs and how to use them in
	// policies, see IAM Identifiers (http://docs.aws.amazon.com/IAM/latest/UserGuide/reference_identifiers.html)
	// in Using IAM.
	//
	// Arn is a required field
	Arn string

	// A unique identifier that contains the role ID and the role session name of
	// the role that is being assumed. The role ID is generated by AWS when the
	// role is created.
	//
	// AssumedRoleId is a required field
	AssumedRoleID string `xml:"AssumeRoleId"`
	// contains filtered or unexported fields
}

// AssumeRoleWithClientGrantsResponse contains the result of successful AssumeRoleWithClientGrants request.
type AssumeRoleWithClientGrantsResponse struct {
	XMLName          xml.Name           `xml:"https://sts.amazonaws.com/doc/2011-06-15/ AssumeRoleWithClientGrantsResponse" json:"-"`
	Result           ClientGrantsResult `xml:"AssumeRoleWithClientGrantsResult"`
	ResponseMetadata struct {
		RequestID string `xml:"RequestId,omitempty"`
	} `xml:"ResponseMetadata,omitempty"`
}

// ClientGrantsResult - Contains the response to a successful AssumeRoleWithClientGrants
// request, including temporary credentials that can be used to make Minio API requests.
type ClientGrantsResult struct {
	// The identifiers for the temporary security credentials that the operation
	// returns.
	AssumedRoleUser AssumedRoleUser `xml:",omitempty"`

	// The intended audience (also known as client ID) of the web identity token.
	// This is traditionally the client identifier issued to the application that
	// requested the client grants.
	Audience string `xml:",omitempty"`

	// The temporary security credentials, which include an access key ID, a secret
	// access key, and a security (or session) token.
	//
	// Note: The size of the security token that STS APIs return is not fixed. We
	// strongly recommend that you make no assumptions about the maximum size. As
	// of this writing, the typical size is less than 4096 bytes, but that can vary.
	// Also, future updates to AWS might require larger sizes.
	Credentials auth.Credentials `xml:",omitempty"`

	// A percentage value that indicates the size of the policy in packed form.
	// The service rejects any policy with a packed size greater than 100 percent,
	// which means the policy exceeded the allowed space.
	PackedPolicySize int `xml:",omitempty"`

	// The issuing authority of the web identity token presented. For OpenID Connect
	// ID tokens, this contains the value of the iss field. For OAuth 2.0 access tokens,
	// this contains the value of the ProviderId parameter that was passed in the
	// AssumeRoleWithClientGrants request.
	Provider string `xml:",omitempty"`

	// The unique user identifier that is returned by the identity provider.
	// This identifier is associated with the Token that was submitted
	// with the AssumeRoleWithClientGrants call. The identifier is typically unique to
	// the user and the application that acquired the ClientGrantsToken (pairwise identifier).
	// For OpenID Connect ID tokens, this field contains the value returned by the identity
	// provider as the token's sub (Subject) claim.
	SubjectFromToken string `xml:",omitempty"`
}

// AssumeRoleWithClientGrants - implementation of AWS STS extension API supporting
// OAuth2.0 client grants.
//
// Eg:-
//    $ curl https://minio:9000/?Action=AssumeRoleWithClientGrants&DurationSeconds=3600&Token=<jwt>
func (sts *stsAPIHandlers) AssumeRoleWithClientGrants(w http.ResponseWriter, r *http.Request) {
	ctx := newContext(r, w, "AssumeRoleWithClientGrants")

	// NOTE: this API only accepts JWT tokens.
	v, err := globalIAMValidators.Get("jwt")
	if err != nil {
		writeSTSErrorResponse(w, ErrSTSInvalidParameterValue)
		return
	}

	m, err := v.Validate(r)
	if err != nil {
		switch err {
		case validator.ErrTokenExpired:
			writeSTSErrorResponse(w, ErrSTSClientGrantsExpiredToken)
		case validator.ErrInvalidDuration:
			writeSTSErrorResponse(w, ErrSTSInvalidParameterValue)
		default:
			logger.LogIf(ctx, err)
			writeSTSErrorResponse(w, ErrSTSInvalidParameterValue)
		}
		return
	}

	secret := globalServerConfig.GetCredential().SecretKey
	cred, err := auth.GetNewCredentialsWithMetadata(m, secret)
	if err != nil {
		logger.LogIf(ctx, err)
		writeSTSErrorResponse(w, ErrSTSInternalError)
		return
	}

	globalIAMSys.SetUser(cred.AccessKey, cred) // Set the newly generated credentials.
	encodedSuccessResponse := encodeResponse(&AssumeRoleWithClientGrantsResponse{
		Result: ClientGrantsResult{Credentials: cred},
	})

	writeSuccessResponseXML(w, encodedSuccessResponse)
}
