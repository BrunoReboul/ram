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
	"log"
	"strings"
	"sync"
	"time"

	"github.com/BrunoReboul/ram/utilities/aut"
	"github.com/BrunoReboul/ram/utilities/cai"
	"github.com/BrunoReboul/ram/utilities/ffo"
	"github.com/BrunoReboul/ram/utilities/gfs"
	"github.com/BrunoReboul/ram/utilities/glo"
	"github.com/BrunoReboul/ram/utilities/gps"
	"github.com/BrunoReboul/ram/utilities/solution"
	"github.com/google/uuid"
	"google.golang.org/api/option"

	"cloud.google.com/go/firestore"
	"cloud.google.com/go/functions/metadata"
	"cloud.google.com/go/pubsub"
	admin "google.golang.org/api/admin/directory/v1"
)

const waitSecOnQuotaExceeded = 70

// Global variable to deal with GroupsListCall Pages constraint: no possible to pass variable to the function in pages()
// https://pkg.go.dev/google.golang.org/api/admin/directory/v1?tab=doc#GroupsListCall.Pages
var ctx context.Context
var directoryCustomerID string
var domain string
var emailPrefix string
var environment string
var instanceName string
var logEventEveryXPubSubMsg uint64
var microserviceName string
var outputTopicName string
var pubSubClient *pubsub.Client
var pubSubErrNumber uint64
var pubSubID string
var pubSubMsgNumber uint64
var stepStack glo.Steps

// Global structure for global variables to optimize the cloud function performances
type Global struct {
	ctx                     context.Context
	dirAdminService         *admin.Service
	directoryCustomerID     string
	environment             string
	firestoreClient         *firestore.Client
	inputTopicName          string
	instanceName            string
	logEventEveryXPubSubMsg uint64
	maxResultsPerPage       int64 // API Max = 200
	microserviceName        string
	outputTopicName         string
	pubSubClient            *pubsub.Client
	PubSubID                string
	retryTimeOutSeconds     int64
	step                    glo.Step
	stepStack               glo.Steps
}

// Settings from PubSub triggering event
type Settings struct {
	DirectoryCustomerID string    `json:"directoryCustomerID"`
	Domain              string    `json:"domain"`
	EmailPrefix         string    `json:"emailPrefix"`
	StepStack           glo.Steps `json:"step_stack,omitempty"`
}

// Initialize is to be executed in the init() function of the cloud function to optimize the cold start
func Initialize(ctx context.Context, global *Global) (err error) {
	log.SetFlags(0)
	global.ctx = ctx

	var instanceDeployment InstanceDeployment
	var clientOption option.ClientOption
	var ok bool

	initID := fmt.Sprintf("%v", uuid.New())
	err = ffo.ReadUnmarshalYAML(solution.PathToFunctionCode+solution.SettingsFileName, &instanceDeployment)
	if err != nil {
		log.Println(glo.Entry{
			Severity:    "CRITICAL",
			Message:     "init_failed",
			Description: fmt.Sprintf("ReadUnmarshalYAML %s %v", solution.SettingsFileName, err),
			InitID:      initID,
		})
		return err
	}

	global.environment = instanceDeployment.Core.EnvironmentName
	global.instanceName = instanceDeployment.Core.InstanceName
	global.microserviceName = instanceDeployment.Core.ServiceName

	log.Println(glo.Entry{
		MicroserviceName: global.microserviceName,
		InstanceName:     global.instanceName,
		Environment:      global.environment,
		Severity:         "NOTICE",
		Message:          "coldstart",
		InitID:           initID,
	})

	gciAdminUserToImpersonate := instanceDeployment.Settings.Instance.GCI.SuperAdminEmail
	global.directoryCustomerID = instanceDeployment.Settings.Instance.GCI.DirectoryCustomerID
	global.inputTopicName = instanceDeployment.Artifacts.TopicName
	global.logEventEveryXPubSubMsg = instanceDeployment.Settings.Service.LogEventEveryXPubSubMsg
	global.maxResultsPerPage = instanceDeployment.Settings.Service.MaxResultsPerPage
	global.outputTopicName = instanceDeployment.Artifacts.OutputTopicName
	global.retryTimeOutSeconds = instanceDeployment.Settings.Service.GCF.RetryTimeOutSeconds
	projectID := instanceDeployment.Core.SolutionSettings.Hosting.ProjectID
	keyJSONFilePath := solution.PathToFunctionCode + instanceDeployment.Settings.Service.KeyJSONFileName
	serviceAccountEmail := fmt.Sprintf("%s@%s.iam.gserviceaccount.com",
		instanceDeployment.Core.ServiceName,
		instanceDeployment.Core.SolutionSettings.Hosting.ProjectID)

	global.firestoreClient, err = firestore.NewClient(global.ctx, projectID)
	if err != nil {
		log.Println(glo.Entry{
			MicroserviceName: global.microserviceName,
			InstanceName:     global.instanceName,
			Environment:      global.environment,
			Severity:         "CRITICAL",
			Message:          "init_failed",
			Description:      fmt.Sprintf("firestore.NewClient %v", err),
			InitID:           initID,
		})
		return err
	}

	serviceAccountKeyNames, err := gfs.ListKeyNames(ctx, global.firestoreClient, instanceDeployment.Core.ServiceName)
	if err != nil {
		log.Println(glo.Entry{
			MicroserviceName: global.microserviceName,
			InstanceName:     global.instanceName,
			Environment:      global.environment,
			Severity:         "CRITICAL",
			Message:          "init_failed",
			Description:      fmt.Sprintf("gfs.ListKeyNames %v", err),
			InitID:           initID,
		})
		return err
	}

	if clientOption, ok = aut.GetClientOptionAndCleanKeys(ctx,
		serviceAccountEmail,
		keyJSONFilePath,
		instanceDeployment.Core.SolutionSettings.Hosting.ProjectID,
		gciAdminUserToImpersonate,
		[]string{admin.AdminDirectoryGroupReadonlyScope, admin.AdminDirectoryDomainReadonlyScope},
		serviceAccountKeyNames,
		initID,
		global.microserviceName,
		global.instanceName,
		global.environment); !ok {
		return fmt.Errorf("aut.GetClientOptionAndCleanKeys")
	}
	global.dirAdminService, err = admin.NewService(ctx, clientOption)
	if err != nil {
		log.Println(glo.Entry{
			MicroserviceName: global.microserviceName,
			InstanceName:     global.instanceName,
			Environment:      global.environment,
			Severity:         "CRITICAL",
			Message:          "init_failed",
			Description:      fmt.Sprintf("admin.NewService %v", err),
			InitID:           initID,
		})
		return err
	}
	global.pubSubClient, err = pubsub.NewClient(ctx, projectID)
	if err != nil {
		log.Println(glo.Entry{
			MicroserviceName: global.microserviceName,
			InstanceName:     global.instanceName,
			Environment:      global.environment,
			Severity:         "CRITICAL",
			Message:          "init_failed",
			Description:      fmt.Sprintf("pubsub.NewClient %v", err),
			InitID:           initID,
		})
		return err
	}
	return nil
}

// EntryPoint is the function to be executed for each cloud function occurence
func EntryPoint(ctxEvent context.Context, PubSubMessage gps.PubSubMessage, global *Global) error {
	// log.Println(string(PubSubMessage.Data))
	metadata, err := metadata.FromContext(ctxEvent)
	if err != nil {
		// Assume an error on the function invoker and try again.
		log.Println(glo.Entry{
			MicroserviceName:   global.microserviceName,
			InstanceName:       global.instanceName,
			Environment:        global.environment,
			Severity:           "CRITICAL",
			Message:            "redo_on_transient",
			Description:        fmt.Sprintf("pubsub_id no available metadata.FromContext: %v", err),
			TriggeringPubsubID: global.PubSubID,
		})
		return err
	}
	global.stepStack = nil
	global.PubSubID = metadata.EventID
	parts := strings.Split(metadata.Resource.Name, "/")
	global.step = glo.Step{
		StepID:        fmt.Sprintf("%s/%s", parts[len(parts)-1], global.PubSubID),
		StepTimestamp: metadata.Timestamp,
	}

	now := time.Now()
	d := now.Sub(metadata.Timestamp)
	log.Println(glo.Entry{
		MicroserviceName:           global.microserviceName,
		InstanceName:               global.instanceName,
		Environment:                global.environment,
		Severity:                   "NOTICE",
		Message:                    "start",
		TriggeringPubsubID:         global.PubSubID,
		TriggeringPubsubAgeSeconds: d.Seconds(),
		TriggeringPubsubTimestamp:  &metadata.Timestamp,
		Now:                        &now,
	})

	if d.Seconds() > float64(global.retryTimeOutSeconds) {
		log.Println(glo.Entry{
			MicroserviceName:           global.microserviceName,
			InstanceName:               global.instanceName,
			Environment:                global.environment,
			Severity:                   "CRITICAL",
			Message:                    "noretry",
			Description:                "Pubsub message too old",
			TriggeringPubsubID:         global.PubSubID,
			TriggeringPubsubAgeSeconds: d.Seconds(),
			TriggeringPubsubTimestamp:  &metadata.Timestamp,
			Now:                        &now,
		})
		return nil
	}

	// Pass data to global variables to deal with func browseGroup
	ctx = global.ctx
	directoryCustomerID = global.directoryCustomerID
	logEventEveryXPubSubMsg = global.logEventEveryXPubSubMsg
	pubSubClient = global.pubSubClient
	outputTopicName = global.outputTopicName
	pubSubID = global.PubSubID
	microserviceName = global.microserviceName
	instanceName = global.instanceName
	environment = global.environment

	if strings.HasPrefix(string(PubSubMessage.Data), "cron schedule") {
		global.stepStack = append(global.stepStack, global.step)

		err = initiateQueries(global)
		if err != nil {
			log.Println(glo.Entry{
				MicroserviceName:   global.microserviceName,
				InstanceName:       global.instanceName,
				Environment:        global.environment,
				Severity:           "CRITICAL",
				Message:            "redo_on_transient",
				Description:        fmt.Sprintf("initiateQueries %v", err),
				TriggeringPubsubID: global.PubSubID,
			})
			return err
		}
		now := time.Now()
		latency := now.Sub(metadata.Timestamp)
		latencyE2E := now.Sub(global.stepStack[0].StepTimestamp)
		log.Println(glo.Entry{
			MicroserviceName:     global.microserviceName,
			InstanceName:         global.instanceName,
			Environment:          global.environment,
			Severity:             "NOTICE",
			Message:              "finish launching sub queries",
			Description:          "Pubsub messages published to reentrant topic to initiate sub queries",
			Now:                  &now,
			TriggeringPubsubID:   global.PubSubID,
			OriginEventTimestamp: &metadata.Timestamp,
			LatencySeconds:       latency.Seconds(),
			LatencyE2ESeconds:    latencyE2E.Seconds(),
			StepStack:            global.stepStack,
			AssetInventoryOrigin: "batch-listgroups",
		})
	} else {
		var settings Settings
		err = json.Unmarshal(PubSubMessage.Data, &settings)
		if err != nil {
			log.Println(glo.Entry{
				MicroserviceName:   global.microserviceName,
				InstanceName:       global.instanceName,
				Environment:        global.environment,
				Severity:           "CRITICAL",
				Message:            "noretry",
				Description:        fmt.Sprintf("json.Unmarshal(PubSubMessage.Data, &settings) %v %v", PubSubMessage.Data, err),
				TriggeringPubsubID: global.PubSubID,
			})
			return nil
		}
		if settings.DirectoryCustomerID != directoryCustomerID {
			log.Println(glo.Entry{
				MicroserviceName:   global.microserviceName,
				InstanceName:       global.instanceName,
				Environment:        global.environment,
				Severity:           "INFO",
				Message:            "ignore this trigerring event",
				Description:        fmt.Sprintf("as directoryCustomerID %s not equal to this instance directoryCustomerID %s", settings.DirectoryCustomerID, directoryCustomerID),
				TriggeringPubsubID: global.PubSubID,
			})
		} else {
			domain = settings.Domain
			emailPrefix = settings.EmailPrefix
			if settings.StepStack != nil {
				global.stepStack = append(settings.StepStack, global.step)
			} else {
				global.stepStack = append(global.stepStack, global.step)
			}
			stepStack = global.stepStack // as a global variable used in the browse function

			err = queryDirectory(settings.Domain, settings.EmailPrefix, global)
			if err != nil {
				if strings.Contains(strings.ToLower(err.Error()), "quota") {
					log.Println(glo.Entry{
						MicroserviceName:   global.microserviceName,
						InstanceName:       global.instanceName,
						Environment:        global.environment,
						Severity:           "WARNING",
						Message:            fmt.Sprintf("waiting_on_quota_exceeded"),
						Description:        fmt.Sprintf("ListGroup quota is gone, wait for %d seconds then retry", waitSecOnQuotaExceeded),
						TriggeringPubsubID: global.PubSubID,
					})
					time.Sleep(waitSecOnQuotaExceeded * time.Second)
					return err
				}
				log.Println(glo.Entry{
					MicroserviceName:   global.microserviceName,
					InstanceName:       global.instanceName,
					Environment:        global.environment,
					Severity:           "CRITICAL",
					Message:            "redo_on_transient",
					Description:        fmt.Sprintf("queryDirectory %v", err),
					TriggeringPubsubID: global.PubSubID,
				})
				return err
			}
		}
	}
	return nil
}

func initiateQueries(global *Global) error {
	figures := getByteSet('0', 10)
	alphabetLower := getByteSet('a', 26)

	emailAuthorizedByteSet := append(figures, alphabetLower...)
	log.Println(glo.Entry{
		MicroserviceName:   global.microserviceName,
		InstanceName:       global.instanceName,
		Environment:        global.environment,
		Severity:           "INFO",
		Message:            "initiate multiple queries",
		Description:        fmt.Sprintf("emailAuthorizedByteSet %s", string(emailAuthorizedByteSet)),
		TriggeringPubsubID: global.PubSubID,
	})

	domains, err := global.dirAdminService.Domains.List(global.directoryCustomerID).Context(global.ctx).Do()
	if err != nil {
		return fmt.Errorf("dirAdminService.Domains.List: %v", err)
	}
	for _, domain := range domains.Domains {
		for _, emailPrefix := range emailAuthorizedByteSet {
			var settings Settings
			settings.DirectoryCustomerID = global.directoryCustomerID
			settings.Domain = domain.DomainName
			settings.EmailPrefix = string(emailPrefix)
			settings.StepStack = global.stepStack
			settingsJSON, err := json.Marshal(settings)
			if err != nil {
				log.Println(glo.Entry{
					MicroserviceName:   global.microserviceName,
					InstanceName:       global.instanceName,
					Environment:        global.environment,
					Severity:           "WARNING",
					Message:            "json.Marshal(settings)",
					Description:        fmt.Sprintf("settings %v", settings),
					TriggeringPubsubID: global.PubSubID,
				})
			} else {
				pubSubMessage := &pubsub.Message{
					Data: settingsJSON,
				}
				topic := global.pubSubClient.Topic(global.inputTopicName)
				id, err := topic.Publish(global.ctx, pubSubMessage).Get(global.ctx)
				if err != nil {
					log.Println(glo.Entry{
						MicroserviceName:   global.microserviceName,
						InstanceName:       global.instanceName,
						Environment:        global.environment,
						Severity:           "WARNING",
						Message:            "topic.Publish",
						Description:        fmt.Sprintf("pubSubMessage %v", pubSubMessage),
						TriggeringPubsubID: global.PubSubID,
					})
				} else {
					log.Println(glo.Entry{
						MicroserviceName:   global.microserviceName,
						InstanceName:       global.instanceName,
						Environment:        global.environment,
						Severity:           "INFO",
						Message:            "Pubsub msg published to reentrant topic",
						Description:        fmt.Sprintf("initiate sub query: domain '%s' emailPrefix '%s' to topic %s msg id: %s", settings.Domain, settings.EmailPrefix, global.inputTopicName, id),
						TriggeringPubsubID: global.PubSubID,
					})
				}
			}
		}
	}
	return nil
}

func queryDirectory(domain string, emailPrefix string, global *Global) error {
	log.Println(glo.Entry{
		MicroserviceName:   global.microserviceName,
		InstanceName:       global.instanceName,
		Environment:        global.environment,
		Severity:           "INFO",
		Message:            fmt.Sprintf("settings retrieved, launch query on domain '%s' and email prefix '%s'", domain, emailPrefix),
		TriggeringPubsubID: global.PubSubID,
	})
	pubSubMsgNumber = 0
	pubSubErrNumber = 0
	query := fmt.Sprintf("email:%s*", emailPrefix)
	// log.Printf("query: %s", query)
	// pages function expect just the name of the callback function. Not an invocation of the function
	err := global.dirAdminService.Groups.List().Customer(global.directoryCustomerID).Domain(domain).Query(query).MaxResults(global.maxResultsPerPage).OrderBy("email").Pages(global.ctx, browseGroups)
	if err != nil {
		if strings.Contains(err.Error(), "Domain not found") {
			now := time.Now()
			latency := now.Sub(global.step.StepTimestamp)
			latencyE2E := now.Sub(global.stepStack[0].StepTimestamp)
			log.Println(glo.Entry{
				MicroserviceName:     global.microserviceName,
				InstanceName:         global.instanceName,
				Environment:          global.environment,
				Severity:             "NOTICE",
				Message:              "cancel",
				Description:          fmt.Sprintf("domain not found %s query %s customer ID %s", domain, query, global.directoryCustomerID),
				Now:                  &now,
				TriggeringPubsubID:   global.PubSubID,
				OriginEventTimestamp: &global.stepStack[0].StepTimestamp,
				LatencySeconds:       latency.Seconds(),
				LatencyE2ESeconds:    latencyE2E.Seconds(),
				StepStack:            global.stepStack,
			})
		} else {
			return fmt.Errorf("dirAdminService.Groups.List: %v", err)
		}
	}
	if pubSubMsgNumber > 0 {
		now := time.Now()
		latency := now.Sub(global.step.StepTimestamp)
		latencyE2E := now.Sub(global.stepStack[0].StepTimestamp)
		log.Println(glo.Entry{
			MicroserviceName:     global.microserviceName,
			InstanceName:         global.instanceName,
			Environment:          global.environment,
			Severity:             "NOTICE",
			Message:              fmt.Sprintf("finish %d groups", pubSubMsgNumber),
			Description:          fmt.Sprintf("directory %s domain '%s' emailPrefix '%s' Number of groups published %d to topic %s", directoryCustomerID, domain, emailPrefix, pubSubMsgNumber, outputTopicName),
			Now:                  &now,
			TriggeringPubsubID:   global.PubSubID,
			OriginEventTimestamp: &global.stepStack[0].StepTimestamp,
			LatencySeconds:       latency.Seconds(),
			LatencyE2ESeconds:    latencyE2E.Seconds(),
			StepStack:            global.stepStack,
			AssetInventoryOrigin: "batch-listgroups",
		})
	} else {
		now := time.Now()
		latency := now.Sub(global.step.StepTimestamp)
		latencyE2E := now.Sub(global.stepStack[0].StepTimestamp)
		log.Println(glo.Entry{
			MicroserviceName:     global.microserviceName,
			InstanceName:         global.instanceName,
			Environment:          global.environment,
			Severity:             "NOTICE",
			Message:              "cancel",
			Description:          fmt.Sprintf("no group found for directory %s domain '%s' emailPrefix '%s'", directoryCustomerID, domain, emailPrefix),
			Now:                  &now,
			TriggeringPubsubID:   global.PubSubID,
			OriginEventTimestamp: &global.stepStack[0].StepTimestamp,
			LatencySeconds:       latency.Seconds(),
			LatencyE2ESeconds:    latencyE2E.Seconds(),
			StepStack:            global.stepStack,
		})
	}
	if pubSubErrNumber > 0 {
		log.Println(glo.Entry{
			MicroserviceName:   global.microserviceName,
			InstanceName:       global.instanceName,
			Environment:        global.environment,
			Severity:           "WARNING",
			Message:            "some pubsub messages did not publish successfully",
			Description:        fmt.Sprintf("number of droped pussub messages %d", pubSubErrNumber),
			TriggeringPubsubID: pubSubID,
		})
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
		var feedMessage cai.FeedMessageGroup
		feedMessage.Window.StartTime = time.Now()
		feedMessage.Origin = "batch-listgroups"
		feedMessage.Deleted = false
		feedMessage.Asset.Ancestors = []string{fmt.Sprintf("directories/%s", directoryCustomerID)}
		feedMessage.Asset.AncestryPath = fmt.Sprintf("directories/%s", directoryCustomerID)
		feedMessage.Asset.AssetType = "www.googleapis.com/admin/directory/groups"
		feedMessage.Asset.Name = fmt.Sprintf("//directories/%s/groups/%s", directoryCustomerID, group.Id)
		feedMessage.Asset.Resource = group
		feedMessage.Asset.Resource.Etag = ""
		feedMessage.StepStack = stepStack
		feedMessageJSON, err := json.Marshal(feedMessage)
		if err != nil {
			log.Println(glo.Entry{
				MicroserviceName:   microserviceName,
				InstanceName:       instanceName,
				Environment:        environment,
				Severity:           "WARNING",
				Message:            "json.Marshal(settings)",
				Description:        fmt.Sprintf("feedMessage %v", feedMessage),
				TriggeringPubsubID: pubSubID,
			})
		} else {
			pubSubMessage := &pubsub.Message{
				Data: feedMessageJSON,
			}
			publishResult := topic.Publish(ctx, pubSubMessage)
			waitgroup.Add(1)
			go gps.GetPublishCallResult(ctx,
				publishResult,
				&waitgroup,
				directoryCustomerID+"/"+group.Email,
				&pubSubErrNumber,
				&pubSubMsgNumber,
				logEventEveryXPubSubMsg,
				pubSubID,
				microserviceName,
				instanceName,
				environment)
		}
	}
	waitgroup.Wait()
	return nil
}

// getByteSet return a set of lenght contiguous bytes starting at bytes
func getByteSet(start byte, length int) []byte {
	byteSet := make([]byte, length)
	for i := range byteSet {
		byteSet[i] = start + byte(i)
	}
	return byteSet
}
