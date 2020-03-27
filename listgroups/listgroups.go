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

package listgroups

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/BrunoReboul/ram/helper"
	"golang.org/x/oauth2/google"
	"golang.org/x/oauth2/jwt"
	"google.golang.org/api/iam/v1"
	"google.golang.org/api/option"

	"cloud.google.com/go/pubsub"
	admin "google.golang.org/api/admin/directory/v1"
)

// Global variable to deal with GroupsListCall Pages constraint: no possible to pass variable to the function in pages()
// https://pkg.go.dev/google.golang.org/api/admin/directory/v1?tab=doc#GroupsListCall.Pages
var ctx context.Context
var directoryCustomerID string
var domain string
var emailPrefix string
var logEventEveryXPubSubMsg uint64
var pubSubClient *pubsub.Client
var outputTopicName string
var pubSubErrNumber uint64
var pubSubMsgNumber uint64
var timestamp time.Time

// Global structure for global variables to optimize the cloud function performances
type Global struct {
	ctx                     context.Context
	dirAdminService         *admin.Service
	directoryCustomerID     string
	initFailed              bool
	inputTopicName          string
	logEventEveryXPubSubMsg uint64
	maxResultsPerPage       int64 // API Max = 200
	outputTopicName         string
	pubSubClient            *pubsub.Client
	retryTimeOutSeconds     int64
}

// FeedMessage Cloud Asset Inventory feed message
type FeedMessage struct {
	Asset   Asset         `json:"asset"`
	Window  helper.Window `json:"window"`
	Deleted bool          `json:"deleted"`
	Origin  string        `json:"origin"`
}

// Asset uses the new CAI feed format
type Asset struct {
	Name         string          `json:"name"`
	AssetType    string          `json:"assetType"`
	Ancestors    []string        `json:"ancestors"`
	AncestryPath string          `json:"ancestryPath"`
	IamPolicy    json.RawMessage `json:"iamPolicy"`
	Resource     *admin.Group    `json:"resource"`
}

// Settings from PubSub triggering event
type Settings struct {
	Domain      string `json:"domain"`
	EmailPrefix string `json:"emailPrefix"`
}

// Initialize is to be executed in the init() function of the cloud function to optimize the cold start
func Initialize(ctx context.Context, global *Global) {
	global.ctx = ctx
	global.initFailed = false

	// err is pre-declared to avoid shadowing client.
	var currentKeyName string
	var err error
	var gciAdminUserToImpersonate string
	var iamService *iam.Service
	var keyJSONFilePath string
	var projectID string
	var serviceAccountEmail string

	gciAdminUserToImpersonate = os.Getenv("GCIADMINUSERTOIMPERSONATE")
	global.directoryCustomerID = os.Getenv("DIRECTORYCUSTOMERID")
	global.inputTopicName = os.Getenv("INPUTTOPICNAME")
	global.outputTopicName = os.Getenv("OUTPUTTOPICNAME")
	keyJSONFilePath = "./" + os.Getenv("KEYJSONFILENAME")
	projectID = os.Getenv("GCP_PROJECT")
	serviceAccountEmail = os.Getenv("SERVICEACCOUNTNAME")

	log.Println("Function COLD START")
	global.retryTimeOutSeconds, err = strconv.ParseInt(os.Getenv("RETRYTIMEOUTSECONDS"), 10, 64)
	if err != nil {
		log.Printf("ERROR - Env variable RETRYTIMEOUTSECONDS cannot be converted to int64: %v", err)
		global.initFailed = true
		return
	}
	global.logEventEveryXPubSubMsg, err = strconv.ParseUint(os.Getenv("LOGEVENTEVERYXPUBSUBMSG"), 10, 64)
	if err != nil {
		log.Printf("Env variable LOGEVENTEVERYXPUBSUBMSG cannot be converted to uint64: %v", err)
		global.initFailed = true
		return
	}
	// log.Printf("logEventEveryXPubSubMsg %d", logEventEveryXPubSubMsg)
	global.maxResultsPerPage, err = strconv.ParseInt(os.Getenv("MAXRESULTSPERPAGE"), 10, 64)
	if err != nil {
		log.Printf("Env variable MAXRESULTSPERPAGE cannot be converted to int: %v", err)
		global.initFailed = true
		return
	}
	iamService, err = iam.NewService(ctx)
	if err != nil {
		log.Printf("ERROR - iam.NewService: %v", err)
		global.initFailed = true
		return
	}
	resource := "projects/-/serviceAccounts/" + serviceAccountEmail
	response, err := iamService.Projects.ServiceAccounts.Keys.List(resource).Do()
	if err != nil {
		log.Printf("ERROR - iamService.Projects.ServiceAccounts.Keys.List: %v", err)
		global.initFailed = true
		return
	}
	keyJSONdata, err := ioutil.ReadFile(keyJSONFilePath)
	if err != nil {
		log.Printf("ERROR - ioutil.ReadFile(keyJSONFilePath): %v", err)
		global.initFailed = true
		return
	}
	var key helper.Key
	err = json.Unmarshal(keyJSONdata, &key)
	if err != nil {
		log.Printf("ERROR - json.Unmarshal(keyJSONdata, &key): %v", err)
		global.initFailed = true
		return
	}
	currentKeyName = "projects/" + projectID + "/serviceAccounts/" + serviceAccountEmail + "/keys/" + key.PrivateKeyID

	// Clean keys
	for _, key := range response.Keys {
		if key.Name == currentKeyName {
			log.Printf("Keep key ValidAfterTime %s named %s", key.ValidAfterTime, key.Name)
		} else {
			if key.KeyType == "SYSTEM_MANAGED" {
				log.Printf("Ignore SYSTEM_MANAGED key named %s", key.Name)
			} else {
				log.Printf("Delete KeyType %s ValidAfterTime %s key name %s", key.KeyType, key.ValidAfterTime, key.Name)
				_, err = iamService.Projects.ServiceAccounts.Keys.Delete(key.Name).Do()
				if err != nil {
					log.Printf("ERROR - iamService.Projects.ServiceAccounts.Keys.Delete: %v", err)
					global.initFailed = true
					return
				}
			}
		}
	}

	// using Json Web joken a the method with cerdentials does not yet implement the subject impersonification
	// https://github.com/googleapis/google-api-java-client/issues/1007

	var jwtConfig *jwt.Config
	// scope constants: https://godoc.org/google.golang.org/api/admin/directory/v1#pkg-constants
	jwtConfig, err = google.JWTConfigFromJSON(keyJSONdata, admin.AdminDirectoryGroupReadonlyScope, admin.AdminDirectoryDomainReadonlyScope)
	if err != nil {
		log.Printf("google.JWTConfigFromJSON: %v", err)
		global.initFailed = true
		return
	}
	jwtConfig.Subject = gciAdminUserToImpersonate
	// jwtConfigJSON, err := json.Marshal(jwtConfig)
	// log.Printf("jwt %s", string(jwtConfigJSON))

	httpClient := jwtConfig.Client(ctx)

	// Use client option as admin.New(httpClient) is deprecated https://godoc.org/google.golang.org/api/admin/directory/v1#New
	var clientOption option.ClientOption
	clientOption = option.WithHTTPClient(httpClient)

	global.dirAdminService, err = admin.NewService(ctx, clientOption)
	if err != nil {
		log.Printf("ERROR - admin.NewService: %v", err)
		global.initFailed = true
		return
	}

	global.pubSubClient, err = pubsub.NewClient(ctx, projectID)
	if err != nil {
		log.Printf("ERROR - pubsub.NewClient: %v", err)
		global.initFailed = true
		return
	}
}

// EntryPoint is the function to be executed for each cloud function occurence
func EntryPoint(ctxEvent context.Context, PubSubMessage helper.PubSubMessage, global *Global) error {
	// log.Println(string(PubSubMessage.Data))
	ok, metadata, err := helper.IntialRetryCheck(ctxEvent, global.initFailed, global.retryTimeOutSeconds)
	if !ok {
		return err
	}
	// log.Printf("EventType %s EventID %s Resource %s Timestamp %v", metadata.EventType, metadata.EventID, metadata.Resource.Type, metadata.Timestamp)

	// Pass data to global variables to deal with func browseGroup
	ctx = global.ctx
	directoryCustomerID = global.directoryCustomerID
	logEventEveryXPubSubMsg = global.logEventEveryXPubSubMsg
	pubSubClient = global.pubSubClient
	outputTopicName = global.outputTopicName
	timestamp = metadata.Timestamp

	if strings.HasPrefix(string(PubSubMessage.Data), "cron schedule") {
		err = initiateQueries(global)
		if err != nil {
			return fmt.Errorf("initiateQueries: %v", err)
		}
	} else {
		var settings Settings
		err = json.Unmarshal(PubSubMessage.Data, &settings)
		if err != nil {
			return fmt.Errorf("json.Unmarshal(PubSubMessage.Data, &settings) %v", err)
		}
		domain = settings.Domain
		emailPrefix = settings.EmailPrefix
		err = queryDirectory(settings.Domain, settings.EmailPrefix, global)
		if err != nil {
			return fmt.Errorf("queryDirectory: %v", err)
		}
	}
	return nil
}

func initiateQueries(global *Global) error {
	figures := helper.GetByteSet('0', 10)
	alphabetLower := helper.GetByteSet('a', 26)
	// Query on directory group email is NOT case sensitive
	// alphabetUpper := getByteSet('A', 26)

	emailAuthorizedByteSet := append(figures, alphabetLower...)
	// emailAuthorizedByteSet := append(emailAuthorizedByteSet, alphabetUpper...)
	log.Printf("Initiate multiple queries on emailAuthorizedByteSet: %s", string(emailAuthorizedByteSet))

	domains, err := global.dirAdminService.Domains.List(global.directoryCustomerID).Context(global.ctx).Do()
	if err != nil {
		return fmt.Errorf("dirAdminService.Domains.List: %v", err) // RETRY
	}
	for _, domain := range domains.Domains {
		for _, emailPrefix := range emailAuthorizedByteSet {
			var settings Settings
			settings.Domain = domain.DomainName
			settings.EmailPrefix = string(emailPrefix)
			settingsJSON, err := json.Marshal(settings)
			if err != nil {
				log.Printf("ERROR - json.Marshal(settings) %v", err) // NO RETRY
			}
			pubSubMessage := &pubsub.Message{
				Data: settingsJSON,
			}
			topic := global.pubSubClient.Topic(global.inputTopicName)
			id, err := topic.Publish(global.ctx, pubSubMessage).Get(global.ctx)
			if err != nil {
				log.Printf("ERROR - pubSubClient.Topic initateQuery: %v", err) // NO RETRY
			}
			log.Printf("Initiate query domain '%s' emailPrefix '%s' to topic %s msg id: %s", settings.Domain, settings.EmailPrefix, global.inputTopicName, id)
		}
	}
	return nil
}

func queryDirectory(domain string, emailPrefix string, global *Global) error {
	log.Printf("Settings retrieved, launch query on domain '%s' and email prefix '%s'", domain, emailPrefix)
	pubSubMsgNumber = 0
	pubSubErrNumber = 0
	query := fmt.Sprintf("email:%s*", emailPrefix)
	log.Printf("query: %s", query)
	// pages function expect just the name of the callback function. Not an invocation of the function
	err := global.dirAdminService.Groups.List().Customer(global.directoryCustomerID).Domain(domain).Query(query).MaxResults(global.maxResultsPerPage).OrderBy("email").Pages(global.ctx, browseGroups)
	if err != nil {
		if strings.Contains(err.Error(), "Domain not found") {
			log.Printf("INFO - Domain not found %s query %s customer ID %s", domain, query, global.directoryCustomerID) // NO RETRY
		} else {
			return fmt.Errorf("dirAdminService.Groups.List: %v", err) // RETRY
		}
	}
	if pubSubMsgNumber > 0 {
		log.Printf("Finished - Directory %s domain '%s' emailPrefix '%s' Number of groups published %d to topic %s", directoryCustomerID, domain, emailPrefix, pubSubMsgNumber, outputTopicName)
	} else {
		log.Printf("No group found for directory %s domain '%s' emailPrefix '%s'", directoryCustomerID, domain, emailPrefix)
	}
	if pubSubErrNumber > 0 {
		log.Printf("%d messages did not publish successfully", pubSubErrNumber) // NO RETRY
	}
	return nil
}

// browseGroups is executed for each page returning a set of groups
// A non-nil error returned will halt the iteration
// the only accepted parameter is groups: https://pkg.go.dev/google.golang.org/api/admin/directory/v1?tab=doc#GroupsListCall.Pages
// so, it use global variables to this package
func browseGroups(groups *admin.Groups) error {
	var waitgroup sync.WaitGroup
	topic := pubSubClient.Topic(outputTopicName)
	for _, group := range groups.Groups {
		var feedMessage FeedMessage
		feedMessage.Window.StartTime = timestamp
		feedMessage.Origin = "batch-listgroups"
		feedMessage.Deleted = false
		feedMessage.Asset.Ancestors = []string{fmt.Sprintf("directories/%s", directoryCustomerID)}
		feedMessage.Asset.AncestryPath = fmt.Sprintf("directories/%s", directoryCustomerID)
		feedMessage.Asset.AssetType = "www.googleapis.com/admin/directory/groups"
		feedMessage.Asset.Name = fmt.Sprintf("//directories/%s/groups/%s", directoryCustomerID, group.Id)
		feedMessage.Asset.Resource = group
		feedMessage.Asset.Resource.Etag = ""
		feedMessageJSON, err := json.Marshal(feedMessage)
		if err != nil {
			log.Printf("ERROR - %s json.Marshal(feedMessage): %v", group.Email, err)
		} else {
			pubSubMessage := &pubsub.Message{
				Data: feedMessageJSON,
			}
			publishResult := topic.Publish(ctx, pubSubMessage)
			waitgroup.Add(1)
			go helper.GetPublishCallResult(ctx, publishResult, &waitgroup, directoryCustomerID+"/"+group.Email, &pubSubErrNumber, &pubSubMsgNumber, logEventEveryXPubSubMsg)
		}
	}
	waitgroup.Wait()
	return nil
}
