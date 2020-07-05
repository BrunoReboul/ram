// Copyright 2020 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the 'License');
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an 'AS IS' BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package ram avoid code redundancy by grouping types and functions used by other ram packages
package ram

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"cloud.google.com/go/firestore"
	"cloud.google.com/go/functions/metadata"
	"cloud.google.com/go/pubsub"

	"golang.org/x/oauth2/google"
	"golang.org/x/oauth2/jwt"
	admin "google.golang.org/api/admin/directory/v1"
	"google.golang.org/api/cloudresourcemanager/v1"
	cloudresourcemanagerv2 "google.golang.org/api/cloudresourcemanager/v2"
	"google.golang.org/api/groupssettings/v1"
	"google.golang.org/api/iam/v1"
	"google.golang.org/api/option"
)

// AssetGroup CAI like format
type AssetGroup struct {
	Name         string          `json:"name"`
	AssetType    string          `json:"assetType"`
	Ancestors    []string        `json:"ancestors"`
	AncestryPath string          `json:"ancestryPath"`
	IamPolicy    json.RawMessage `json:"iamPolicy"`
	Resource     *admin.Group    `json:"resource"`
}

// AssetGroupSettings CAI like format
type AssetGroupSettings struct {
	Name      string                 `json:"name"`
	AssetType string                 `json:"assetType"`
	Ancestors []string               `json:"ancestors"`
	IamPolicy json.RawMessage        `json:"iamPolicy"`
	Resource  *groupssettings.Groups `json:"resource"`
}

// AssetMember CAI like format
type AssetMember struct {
	Name         string          `json:"name"`
	AssetType    string          `json:"assetType"`
	Ancestors    []string        `json:"ancestors"`
	AncestryPath string          `json:"ancestryPath"`
	IamPolicy    json.RawMessage `json:"iamPolicy"`
	Resource     Member          `json:"resource"`
}

// ComplianceStatus by asset, by rule, true/false compliance status
type ComplianceStatus struct {
	AssetName               string    `json:"assetName"`
	AssetInventoryTimeStamp time.Time `json:"assetInventoryTimeStamp"`
	AssetInventoryOrigin    string    `json:"assetInventoryOrigin"`
	RuleName                string    `json:"ruleName"`
	RuleDeploymentTimeStamp time.Time `json:"ruleDeploymentTimeStamp"`
	Compliant               bool      `json:"compliant"`
	Deleted                 bool      `json:"deleted"`
}

// FeedMessageGroup CAI like format
type FeedMessageGroup struct {
	Asset   AssetGroup `json:"asset"`
	Window  Window     `json:"window"`
	Deleted bool       `json:"deleted"`
	Origin  string     `json:"origin"`
}

// FeedMessageGroupSettings CAI like format
type FeedMessageGroupSettings struct {
	Asset   AssetGroupSettings `json:"asset"`
	Window  Window             `json:"window"`
	Deleted bool               `json:"deleted"`
	Origin  string             `json:"origin"`
}

// FeedMessageMember CAI like format
type FeedMessageMember struct {
	Asset   AssetMember `json:"asset"`
	Window  Window      `json:"window"`
	Deleted bool        `json:"deleted"`
	Origin  string      `json:"origin"`
}

// GCSEvent is the payload of a GCS event.
type GCSEvent struct {
	Kind                    string                 `json:"kind"`
	ID                      string                 `json:"id"`
	SelfLink                string                 `json:"selfLink"`
	Name                    string                 `json:"name"`
	Bucket                  string                 `json:"bucket"`
	Generation              string                 `json:"generation"`
	Metageneration          string                 `json:"metageneration"`
	ContentType             string                 `json:"contentType"`
	TimeCreated             time.Time              `json:"timeCreated"`
	Updated                 time.Time              `json:"updated"`
	TemporaryHold           bool                   `json:"temporaryHold"`
	EventBasedHold          bool                   `json:"eventBasedHold"`
	RetentionExpirationTime time.Time              `json:"retentionExpirationTime"`
	StorageClass            string                 `json:"storageClass"`
	TimeStorageClassUpdated time.Time              `json:"timeStorageClassUpdated"`
	Size                    string                 `json:"size"`
	MD5Hash                 string                 `json:"md5Hash"`
	MediaLink               string                 `json:"mediaLink"`
	ContentEncoding         string                 `json:"contentEncoding"`
	ContentDisposition      string                 `json:"contentDisposition"`
	CacheControl            string                 `json:"cacheControl"`
	Metadata                map[string]interface{} `json:"metadata"`
	CRC32C                  string                 `json:"crc32c"`
	ComponentCount          int                    `json:"componentCount"`
	Etag                    string                 `json:"etag"`
	CustomerEncryption      struct {
		EncryptionAlgorithm string `json:"encryptionAlgorithm"`
		KeySha256           string `json:"keySha256"`
	}
	KMSKeyName    string `json:"kmsKeyName"`
	ResourceState string `json:"resourceState"`
}

// keyConsoleFormat Service account json key using console or gcloud format
type keyConsoleFormat struct {
	Type                    string `json:"type"`
	ProjectID               string `json:"project_id"`
	PrivateKeyID            string `json:"private_key_id"`
	PrivateKey              string `json:"private_key"`
	ClientEmail             string `json:"client_email"`
	ClientID                string `json:"client_id"`
	AuthURI                 string `json:"auth_uri"`
	TokenURI                string `json:"token_uri"`
	AuthProviderX509CertURL string `json:"auth_provider_x509_cert_url"`
	ClientX509CertURL       string `json:"client_x509_cert_url"`
}

// keyRestAPIFormat Service account json key using REST API format
type keyRestAPIFormat struct {
	Name            string `json:"name"`
	PrivateKeyType  string `json:"privateKeyType"`
	PrivateKeyData  string `json:"privateKeyData"`
	ValidAfterTime  string `json:"validAfterTime"`
	ValidBeforeTime string `json:"validBeforeTime"`
	KeyAlgorithm    string `json:"keyAlgorithm"`
}

// Member is sligthly different from admim.Member to have both group email and member email
type Member struct {
	MemberEmail string `json:"memberEmail"`
	GroupEmail  string `json:"groupEmail"`
	ID          string `json:"id"`
	Kind        string `json:"kind"`
	Role        string `json:"role"`
	Type        string `json:"type"`
}

// PublishRequest Pub/sub
type PublishRequest struct {
	Topic string `json:"topic"`
}

// PubSubMessage is the payload of a Pub/Sub event.
type PubSubMessage struct {
	Data []byte `json:"data"`
}

// Window Cloud Asset Inventory feed message time window
type Window struct {
	StartTime time.Time `json:"startTime" firestore:"startTime"`
}

// BuildAncestorsDisplayName build a slice of Ancestor friendly name from a slice of ancestors
func BuildAncestorsDisplayName(ctx context.Context, ancestors []string, collectionID string, firestoreClient *firestore.Client, cloudresourcemanagerService *cloudresourcemanager.Service, cloudresourcemanagerServiceV2 *cloudresourcemanagerv2.Service) []string {
	cnt := len(ancestors)
	ancestorsDisplayName := make([]string, len(ancestors))
	for idx := 0; idx < cnt; idx++ {
		ancestorsDisplayName[idx] = getDisplayName(ctx, ancestors[idx], collectionID, firestoreClient, cloudresourcemanagerService, cloudresourcemanagerServiceV2)
	}
	return ancestorsDisplayName
}

// BuildAncestryPath build a path from a slice of ancestors
func BuildAncestryPath(ancestors []string) string {
	cnt := len(ancestors)
	revAncestors := make([]string, len(ancestors))
	for idx := 0; idx < cnt; idx++ {
		revAncestors[cnt-idx-1] = ancestors[idx]
	}
	var ancestryPath string
	ancestryPath = makeCompatible(strings.Join(revAncestors, "/"))
	return ancestryPath
}

// Find a string in a slice of string. Return true when found else false
func Find(slice []string, val string) bool {
	for _, item := range slice {
		if item == val {
			return true
		}
	}
	return false
}

// FireStoreGetDoc check if a document exist with retries
func FireStoreGetDoc(ctx context.Context, firestoreClient *firestore.Client, documentPath string, retriesNumber time.Duration) (*firestore.DocumentSnapshot, bool) {
	var documentSnap *firestore.DocumentSnapshot
	var err error
	var i time.Duration
	for i = 0; i < retriesNumber; i++ {
		documentSnap, err = firestoreClient.Doc(documentPath).Get(ctx)
		if err != nil {
			log.Printf("ERROR - iteration %d firestoreClient.Doc(documentPath).Get(ctx) %v", i, err)
			time.Sleep(i * 100 * time.Millisecond)
		} else {
			return documentSnap, documentSnap.Exists()
		}
	}
	return documentSnap, false
}

// GetAssetContact retrieve owner of resolver contact from asset labels and parent labels
func GetAssetContact(contactRole string, resourceJSON json.RawMessage) (string, error) {
	var contact string
	var resource struct {
		Data struct {
			Labels map[string]string
		}
	}
	err := json.Unmarshal(resourceJSON, &resource)
	if err != nil {
		return "", err
	}
	if resource.Data.Labels != nil {
		if labelValue, ok := resource.Data.Labels[contactRole]; ok {
			contact = labelValue
		}
	}
	return contact, nil
}

// GetByteSet return a set of lenght contiguous bytes starting at bytes
func GetByteSet(start byte, length int) []byte {
	byteSet := make([]byte, length)
	for i := range byteSet {
		byteSet[i] = start + byte(i)
	}
	return byteSet
}

// getDisplayName retrieive the friendly name of an ancestor
func getDisplayName(ctx context.Context, name string, collectionID string, firestoreClient *firestore.Client, cloudresourcemanagerService *cloudresourcemanager.Service, cloudresourcemanagerServiceV2 *cloudresourcemanagerv2.Service) string {
	var displayName = "unknown"
	ancestorType := strings.Split(name, "/")[0]
	knownAncestorTypes := []string{"organizations", "folders", "projects"}
	if !Find(knownAncestorTypes, ancestorType) {
		return displayName
	}
	documentID := "//cloudresourcemanager.googleapis.com/" + name
	documentID = RevertSlash(documentID)
	documentPath := collectionID + "/" + documentID
	// log.Printf("documentPath:%s", documentPath)
	// documentSnap, err := firestoreClient.Doc(documentPath).Get(ctx)
	documentSnap, found := FireStoreGetDoc(ctx, firestoreClient, documentPath, 10)
	if found {
		assetMap := documentSnap.Data()
		// log.Println(assetMap)
		var assetInterface interface{} = assetMap["asset"]
		if asset, ok := assetInterface.(map[string]interface{}); ok {
			var resourceInterface interface{} = asset["resource"]
			if resource, ok := resourceInterface.(map[string]interface{}); ok {
				var dataInterface interface{} = resource["data"]
				if data, ok := dataInterface.(map[string]interface{}); ok {
					switch ancestorType {
					case "organizations":
						var dNameInterface interface{} = data["displayName"]
						if dName, ok := dNameInterface.(string); ok {
							displayName = dName
						}
					case "folders":
						var dNameInterface interface{} = data["displayName"]
						if dName, ok := dNameInterface.(string); ok {
							displayName = dName
						}
					case "projects":
						var dNameInterface interface{} = data["name"]
						if dName, ok := dNameInterface.(string); ok {
							displayName = dName
						}
					}
				}
			}
		}
		// log.Printf("name %s displayName %s", name, displayName)
	} else {
		log.Printf("WARNING - Not found in firestore %s", documentPath)
		//try resourcemamager API
		switch strings.Split(name, "/")[0] {
		case "organizations":
			resp, err := cloudresourcemanagerService.Organizations.Get(name).Context(ctx).Do()
			if err != nil {
				log.Printf("WARNING - cloudresourcemanagerService.Organizations.Get %v", err)
			} else {
				displayName = resp.DisplayName
			}
		case "folders":
			resp, err := cloudresourcemanagerServiceV2.Folders.Get(name).Context(ctx).Do()
			if err != nil {
				log.Printf("WARNING - cloudresourcemanagerServiceV2.Folders.Get %v", err)
			} else {
				displayName = resp.DisplayName
			}
		case "projects":
			resp, err := cloudresourcemanagerService.Projects.Get(strings.Split(name, "/")[1]).Context(ctx).Do()
			if err != nil {
				log.Printf("WARNING - cloudresourcemanagerService.Projects.Get %v", err)
			} else {
				displayName = resp.Name
			}
		}
	}
	return displayName
}

// GetClientOptionAndCleanKeys build a clientOption object and manage the init state
func GetClientOptionAndCleanKeys(ctx context.Context, serviceAccountEmail string, keyJSONFilePath string, projectID string, gciAdminUserToImpersonate string, scopes []string) (option.ClientOption, bool) {
	var clientOption option.ClientOption
	var jwtConfig *jwt.Config

	jwtConfig, err := GetJWTConfigAndCleanKeys(ctx, serviceAccountEmail, keyJSONFilePath, projectID, gciAdminUserToImpersonate, scopes)
	if err != nil {
		return clientOption, false
	}

	httpClient := jwtConfig.Client(ctx)
	// Use client option as admin.New(httpClient) is deprecated https://godoc.org/google.golang.org/api/admin/directory/v1#New
	clientOption = option.WithHTTPClient(httpClient)

	return clientOption, true
}

// GetJWTConfigAndCleanKeys build a JWT config and manage the init state
func GetJWTConfigAndCleanKeys(ctx context.Context, serviceAccountEmail string, keyJSONFilePath string, projectID string, gciAdminUserToImpersonate string, scopes []string) (jwtConfig *jwt.Config, err error) {
	keyRestAPIFormat, err := getKeyJSONdataAndCleanKeys(ctx, serviceAccountEmail, keyJSONFilePath, projectID)
	if err != nil {
		return jwtConfig, err
	}

	// Convert format
	// https://cloud.google.com/iam/docs/creating-managing-service-account-keys#iam-service-account-keys-create-go
	keyJSONdata, err := base64.StdEncoding.DecodeString(keyRestAPIFormat.PrivateKeyData)
	if err != nil {
		return jwtConfig, err
	}

	//DEBUG
	// var keyConsoleFormat keyConsoleFormat
	// err = json.Unmarshal(keyJSONdata, &keyConsoleFormat)
	// if err != nil {
	// 	return jwtConfig, err
	// }
	// JSONMarshalIndentPrint(keyConsoleFormat)

	// using Json Web joken a the method with cerdentials does not yet implement the subject impersonification
	// https://github.com/googleapis/google-api-java-client/issues/1007
	jwtConfig, err = getJWTConfigAndImpersonate(keyJSONdata, gciAdminUserToImpersonate, scopes)
	if err != nil {
		return jwtConfig, err
	}
	return jwtConfig, nil
}

// getKeyJSONdataAndCleanKeys get the service account key to build a JWT and clean older keys
func getKeyJSONdataAndCleanKeys(ctx context.Context, serviceAccountEmail string, keyJSONFilePath string, projectID string) (keyRestAPIFormat keyRestAPIFormat, err error) {
	var keyJSONdata []byte
	var currentKeyName string
	var iamService *iam.Service

	iamService, err = iam.NewService(ctx)
	if err != nil {
		log.Printf("ERROR - iam.NewService: %v", err)
		return keyRestAPIFormat, err
	}
	resource := "projects/-/serviceAccounts/" + serviceAccountEmail
	listServiceAccountKeyResponse, err := iamService.Projects.ServiceAccounts.Keys.List(resource).Do()
	if err != nil {
		log.Printf("ERROR - iamService.Projects.ServiceAccounts.Keys.List: %v", err)
		return keyRestAPIFormat, err
	}
	keyJSONdata, err = ioutil.ReadFile(keyJSONFilePath)
	if err != nil {
		log.Printf("ERROR - ioutil.ReadFile(keyJSONFilePath): %v", err)
		return keyRestAPIFormat, err
	}
	err = json.Unmarshal(keyJSONdata, &keyRestAPIFormat)
	if err != nil {
		log.Printf("ERROR - json.Unmarshal(keyJSONdata, &keyRestAPIFormat): %v", err)
		return keyRestAPIFormat, err
	}
	currentKeyName = keyRestAPIFormat.Name

	// Clean keys
	for _, serviceAccountKey := range listServiceAccountKeyResponse.Keys {
		if serviceAccountKey.Name == currentKeyName {
			log.Printf("Keep key ValidAfterTime %s named %s", serviceAccountKey.ValidAfterTime, serviceAccountKey.Name)
		} else {
			if serviceAccountKey.KeyType == "SYSTEM_MANAGED" {
				log.Printf("Ignore SYSTEM_MANAGED key named %s", serviceAccountKey.Name)
			} else {
				log.Printf("Delete KeyType %s ValidAfterTime %s key name %s", serviceAccountKey.KeyType, serviceAccountKey.ValidAfterTime, serviceAccountKey.Name)
				_, err = iamService.Projects.ServiceAccounts.Keys.Delete(serviceAccountKey.Name).Do()
				if err != nil {
					log.Printf("ERROR - iamService.Projects.ServiceAccounts.Keys.Delete: %v", err)
					return keyRestAPIFormat, err
				}
			}
		}
	}
	return keyRestAPIFormat, nil
}

// getJWTConfigAndImpersonate build JWT with impersonification
func getJWTConfigAndImpersonate(keyJSONdata []byte, gciAdminUserToImpersonate string, scopes []string) (jwtConfig *jwt.Config, err error) {
	// using Json Web joken a the method with cerdentials does not yet implement the subject impersonification
	// https://github.com/googleapis/google-api-java-client/issues/1007

	// scope constants: https://godoc.org/google.golang.org/api/admin/directory/v1#pkg-constants
	jwtConfig, err = google.JWTConfigFromJSON(keyJSONdata, scopes...)
	if err != nil {
		log.Printf("google.JWTConfigFromJSON: %v", err)
		return jwtConfig, err
	}
	jwtConfig.Subject = gciAdminUserToImpersonate

	// DEBUG
	// jwtConfigJSON, err := json.Marshal(jwtConfig)
	// log.Printf("jwt %s", string(jwtConfigJSON))

	return jwtConfig, nil
}

// GetPublishCallResult func to be used in go routine to scale pubsub event publish
func GetPublishCallResult(ctx context.Context, publishResult *pubsub.PublishResult, waitgroup *sync.WaitGroup, msgInfo string, pubSubErrNumber *uint64, pubSubMsgNumber *uint64, logEventEveryXPubSubMsg uint64) {
	defer waitgroup.Done()
	id, err := publishResult.Get(ctx)
	if err != nil {
		log.Printf("ERROR count %d on %s: %v", atomic.AddUint64(pubSubErrNumber, 1), msgInfo, err)
		return
	}
	msgNumber := atomic.AddUint64(pubSubMsgNumber, 1)
	if msgNumber%logEventEveryXPubSubMsg == 0 {
		// No retry on pubsub publish as already implemented in the GO client
		log.Printf("Progression %d messages published, now %s id %s", msgNumber, msgInfo, id)
	}
	// log.Printf("Progression %d messages published, now %s id %s", msgNumber, msgInfo, id)
}

// IntialRetryCheck performs intitial controls
// 1) return true and metadata when controls are passed
// 2) return false when controls failed:
// - 2a) with an error to retry the cloud function entry point function
// - 2b) with nil to stop the cloud function entry point function
func IntialRetryCheck(ctxEvent context.Context, initFailed bool, retryTimeOutSeconds int64) (bool, *metadata.Metadata, error) {
	metadata, err := metadata.FromContext(ctxEvent)
	if err != nil {
		// Assume an error on the function invoker and try again.
		return false, metadata, fmt.Errorf("metadata.FromContext: %v", err) // RETRY
	}
	if initFailed {
		log.Println("ERROR - init function failed")
		return false, metadata, nil // NO RETRY
	}

	// Ignore events that are too old.
	expiration := metadata.Timestamp.Add(time.Duration(retryTimeOutSeconds) * time.Second)
	if time.Now().After(expiration) {
		log.Printf("ERROR - too many retries for expired event '%q'", metadata.EventID)
		return false, metadata, nil // NO MORE RETRY
	}
	return true, metadata, nil
}

// makeCompatible update a GCP asset ancestryPath to make it compatible with former Policy Library REGO rules
func makeCompatible(path string) string {
	path = strings.Replace(path, "organizations", "organization", -1)
	path = strings.Replace(path, "folders", "folder", -1)
	path = strings.Replace(path, "projects", "project", -1)
	return path
}

// PrintEnptyInterfaceType discover the type below an empty interface
func PrintEnptyInterfaceType(value interface{}, valueName string) error {
	switch t := value.(type) {
	default:
		log.Printf("type %T for value named: %s\n", t, valueName)
	}
	return nil
}

// RevertSlash replace slash / by back slash \
func RevertSlash(txt string) string {
	return strings.Replace(txt, "/", "\\", -1)
}
