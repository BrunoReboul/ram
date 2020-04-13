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

package publish2fs

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/BrunoReboul/ram/utilities/ram"
	"github.com/BrunoReboul/ram/utilities/validater"
	"google.golang.org/api/cloudfunctions/v1"

	"cloud.google.com/go/firestore"
)

const (
	description            = "publish  %s assets resource feeds as FireStore documents in collection %s"
	eventProviderNamespace = "cloud.pubsub"
	eventResourceType      = "topic.publish"
	eventService           = "pubsub.googleapis.com"
)

// Global structure for global variables to optimize the cloud function performances
type Global struct {
	ctx                 context.Context
	initFailed          bool
	retryTimeOutSeconds int64
	collectionID        string
	firestoreClient     *firestore.Client
	solution            ram.SolutionSettings
	service             ServiceSettings
	instance            InstanceSettings
}

// ServiceSettings defines service settings common to all service instances
type ServiceSettings struct {
	GCF struct {
		AvailableMemoryMb   int64  `yaml:"availableMemoryMb"`
		RetryTimeOutSeconds int64  `yaml:"retryTimeOutSeconds"`
		Runtime             string `yaml:"runtime"`
		Timeout             string `yaml:"timeout"`
	}
	Name string `yaml:",omitempty"`
}

// InstanceSettings defines instance specific settings
type InstanceSettings struct {
	GCF struct {
		TriggerTopic string `yaml:"triggerTopic"`
	}
	CloudFunction cloudfunctions.CloudFunction
}

// feedMessage Cloud Asset Inventory feed message
type feedMessage struct {
	Asset   asset      `json:"asset" firestore:"asset"`
	Window  ram.Window `json:"window" firestore:"window"`
	Deleted bool       `json:"deleted" firestore:"deleted"`
	Origin  string     `json:"origin" firestore:"origin"`
}

// Asset Cloud Asset Metadata
type asset struct {
	Name         string                 `json:"name" firestore:"name"`
	AssetType    string                 `json:"assetType" firestore:"assetType"`
	Ancestors    []string               `json:"ancestors" firestore:"ancestors"`
	AncestryPath string                 `json:"ancestryPath" firestore:"ancestryPath"`
	IamPolicy    map[string]interface{} `json:"iamPolicy" firestore:"iamPolicy,omitempty"`
	Resource     map[string]interface{} `json:"resource" firestore:"resource"`
}

// Initialize is to be executed in the init() function of the cloud function to optimize the cold start
func Initialize(ctx context.Context, global *Global) {
	global.ctx = ctx
	global.initFailed = false

	// err is pre-declared to avoid shadowing client.
	var err error
	var ok bool
	var projectID string

	global.collectionID = os.Getenv("COLLECTION_ID")
	projectID = os.Getenv("GCP_PROJECT")

	log.Println("Function COLD START")
	if global.retryTimeOutSeconds, ok = ram.GetEnvVarInt64("RETRYTIMEOUTSECONDS"); !ok {
		return
	}
	global.firestoreClient, err = firestore.NewClient(ctx, projectID)
	if err != nil {
		log.Printf("ERROR - firestore.NewClient: %v", err)
		global.initFailed = true
		return
	}
}

// EntryPoint is the function to be executed for each cloud function occurence
func EntryPoint(ctxEvent context.Context, PubSubMessage ram.PubSubMessage, global *Global) error {
	// log.Println(string(PubSubMessage.Data))
	if ok, _, err := ram.IntialRetryCheck(ctxEvent, global.initFailed, global.retryTimeOutSeconds); !ok {
		return err
	}
	// log.Printf("EventType %s EventID %s Resource %s Timestamp %v", metadata.EventType, metadata.EventID, metadata.Resource.Type, metadata.Timestamp)

	var feedMessage feedMessage
	err := json.Unmarshal(PubSubMessage.Data, &feedMessage)
	if err != nil {
		log.Printf("ERROR - json.Unmarshal: %v", err)
		return nil // NO RETRY
	}
	if feedMessage.Origin == "" {
		feedMessage.Origin = "real-time"
	}
	// log.Printf("%v", feedMessage)

	documentID := ram.RevertSlash(feedMessage.Asset.Name)
	documentPath := global.collectionID + "/" + documentID
	if feedMessage.Deleted == true {
		_, err = global.firestoreClient.Doc(documentPath).Delete(global.ctx)
		if err != nil {
			return fmt.Errorf("Error when deleting %s %v", documentPath, err) // RETRY
		}
		log.Printf("DELETED document: %s", documentPath)
	} else {
		_, err = global.firestoreClient.Doc(documentPath).Set(global.ctx, feedMessage)
		if err != nil {
			return fmt.Errorf("firestoreClient.Doc(documentPath).Set: %s %v", documentPath, err) // RETRY
		}
		log.Printf("SET document: %s", documentPath)
	}
	return nil
}

// ReadConfigFile reads and validates service settings from a config file
func (settings *ServiceSettings) ReadConfigFile(path string) error {
	return ram.ReadUnmarshalYAML(path, settings)
}

// Situate set settings from settings
func (settings *ServiceSettings) Situate(situation interface{}) error {
	if serviceName, ok := situation.(string); ok {
		settings.Name = serviceName
		return nil
	}
	return fmt.Errorf("situation is expected to be the service name")
}

// Validate validates the settings
func (settings *ServiceSettings) Validate() (err error) {
	err = validater.ValidateStruct(settings, "publish2fsServiceSettings")
	if err != nil {
		return err
	}
	return nil
}

// ReadValidateSituate reads settings from a config file, validates then, situates them
func (settings *ServiceSettings) ReadValidateSituate(path string, situation interface{}) (err error) {
	err = settings.ReadConfigFile(path)
	if err != nil {
		return err
	}
	err = settings.Validate()
	if err != nil {
		return err
	}
	err = settings.Situate(situation)
	if err != nil {
		return err
	}
	return nil
}

// ReadConfigFile reads and validates instance settings from a config file
func (settings *InstanceSettings) ReadConfigFile(path string) error {
	return ram.ReadUnmarshalYAML(path, settings)
}

// Validate validates the settings
func (settings *InstanceSettings) Validate() (err error) {
	err = validater.ValidateStruct(settings, "publish2fsInstanceSettings")
	if err != nil {
		return err
	}
	return nil
}

// Situate set settings from settings
func (settings *InstanceSettings) Situate(situation interface{}) error {
	if instanceSituation, ok := situation.(ram.InstanceSituation); ok {
		solutionSettings := instanceSituation.Solution
		if serviceSettings, ok := instanceSituation.Service.(*ServiceSettings); ok {
			var failurePolicy cloudfunctions.FailurePolicy
			retry := cloudfunctions.Retry{}
			failurePolicy.Retry = &retry

			var eventTrigger cloudfunctions.EventTrigger
			eventTrigger.EventType = fmt.Sprintf("providers/%s/eventTypes/%s", eventProviderNamespace, eventResourceType)
			eventTrigger.Resource = fmt.Sprintf("projects/%s/topics/%s", solutionSettings.Hosting.ProjectID, settings.GCF.TriggerTopic)
			eventTrigger.Service = eventService
			eventTrigger.FailurePolicy = &failurePolicy

			settings.CloudFunction.AvailableMemoryMb = serviceSettings.GCF.AvailableMemoryMb
			settings.CloudFunction.Description = fmt.Sprintf(description, settings.GCF.TriggerTopic, solutionSettings.Hosting.FireStore.CollectionIDs.Assets)
			settings.CloudFunction.EntryPoint = "EntryPoint"
			settings.CloudFunction.EnvironmentVariables = nil
			settings.CloudFunction.EventTrigger = &eventTrigger
			settings.CloudFunction.Labels = map[string]string{"name": instanceSituation.InstanceName}
			settings.CloudFunction.Name = fmt.Sprintf("projects/%s/locations/%s/functions/%s", solutionSettings.Hosting.ProjectID, solutionSettings.Hosting.GCF.Region, instanceSituation.InstanceName)
			settings.CloudFunction.Runtime = serviceSettings.GCF.Runtime
			settings.CloudFunction.ServiceAccountEmail = fmt.Sprintf("%s@%s.iam.gserviceaccount.com", serviceSettings.Name, solutionSettings.Hosting.ProjectID)
			settings.CloudFunction.Timeout = serviceSettings.GCF.Timeout
			return nil
		}
		return fmt.Errorf("Provided service settings type in the situation is not the expected type")
	}
	return fmt.Errorf("Provided situation is not the expected type")
}

// ReadValidateSituate reads settings from a config file, validates then, situates them
func (settings *InstanceSettings) ReadValidateSituate(path string, situation interface{}) (err error) {
	err = settings.ReadConfigFile(path)
	if err != nil {
		return err
	}
	err = settings.Validate()
	if err != nil {
		return err
	}
	err = settings.Situate(situation)
	if err != nil {
		return err
	}
	return nil
}

// NewServiceSettings create a service settings structure
func NewServiceSettings() *ServiceSettings {
	return &ServiceSettings{}
}

// NewInstanceSettings create an instance settings structure
func NewInstanceSettings() *InstanceSettings {
	return &InstanceSettings{}
}
