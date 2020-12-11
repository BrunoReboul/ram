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

package stream2bq

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"cloud.google.com/go/functions/metadata"
	"github.com/BrunoReboul/ram/services/monitor"
	"github.com/BrunoReboul/ram/utilities/cai"
	"github.com/BrunoReboul/ram/utilities/ffo"
	"github.com/BrunoReboul/ram/utilities/gbq"
	"github.com/BrunoReboul/ram/utilities/gps"
	"github.com/BrunoReboul/ram/utilities/logging"
	"github.com/BrunoReboul/ram/utilities/solution"
	"github.com/google/uuid"
	"google.golang.org/api/cloudresourcemanager/v1"

	"cloud.google.com/go/bigquery"
	"cloud.google.com/go/firestore"
	cloudresourcemanagerv2 "google.golang.org/api/cloudresourcemanager/v2"
)

// Global structure for global variables to optimize the cloud function performances
type Global struct {
	assetsCollectionID            string
	cloudresourcemanagerService   *cloudresourcemanager.Service
	cloudresourcemanagerServiceV2 *cloudresourcemanagerv2.Service // v2 is needed for folders
	ctx                           context.Context
	environment                   string
	firestoreClient               *firestore.Client
	inserter                      *bigquery.Inserter
	instanceName                  string
	microserviceName              string
	ownerLabelKeyName             string
	PubSubID                      string
	retryTimeOutSeconds           int64
	tableName                     string
	violationResolverLabelKeyName string
}

// violation from the "audit" rego policy in "audit.rego" module
type violation struct {
	NonCompliance    nonCompliance    `json:"nonCompliance"`
	FunctionConfig   functionConfig   `json:"functionConfig"`
	ConstraintConfig constraintConfig `json:"constraintConfig"`
	FeedMessage      feedMessage      `json:"feedMessage"`
	RegoModules      json.RawMessage  `json:"regoModules"`
}

// violationBQ from the "audit" rego policy in "audit.rego" module
type violationBQ struct {
	NonCompliance    nonComplianceBQ    `json:"nonCompliance"`
	FunctionConfig   functionConfig     `json:"functionConfig"`
	ConstraintConfig constraintConfigBQ `json:"constraintConfig"`
	FeedMessage      feedMessageBQ      `json:"feedMessage"`
	RegoModules      string             `json:"regoModules"`
}

// nonCompliance form the "deny" rego policy in a <templateName>.rego module
type nonCompliance struct {
	Message  string          `json:"message"`
	Metadata json.RawMessage `json:"metadata"`
}

// nonComplianceBQ form the "deny" rego policy in a <templateName>.rego module
type nonComplianceBQ struct {
	Message  string `json:"message"`
	Metadata string `json:"metadata"`
}

// functionConfig function deployment settings
type functionConfig struct {
	FunctionName   string    `json:"functionName"`
	DeploymentTime time.Time `json:"deploymentTime"`
	ProjectID      string    `json:"projectID"`
	Environment    string    `json:"environment"`
}

// constraintConfig expose content of the constraint yaml file
type constraintConfig struct {
	APIVersion string             `json:"apiVersion"`
	Kind       string             `json:"kind"`
	Metadata   constraintMetadata `json:"metadata"`
	Spec       spec               `json:"spec"`
}

// constraintConfigBQ format to persist in BQ
type constraintConfigBQ struct {
	Kind     string               `json:"kind"`
	Metadata constraintMetadataBQ `json:"metadata"`
	Spec     specBQ               `json:"spec"`
}

// constraintMetadata Constraint's metadata
type constraintMetadata struct {
	Name       string          `json:"name"`
	Annotation json.RawMessage `json:"annotation"`
}

// constraintMetadataBQ format to persist in BQ
type constraintMetadataBQ struct {
	Name       string `json:"name"`
	Annotation string `json:"annotation"`
}

// spec Constraint's specifications
type spec struct {
	Severity   string          `json:"severity"`
	Match      json.RawMessage `json:"match"`
	Parameters json.RawMessage `json:"parameters"`
}

// specBQ format to persist in BQ
type specBQ struct {
	Severity   string `json:"severity"`
	Match      string `json:"match"`
	Parameters string `json:"parameters"`
}

// feedMessage Cloud Asset Inventory feed message
type feedMessage struct {
	Asset  asset      `json:"asset"`
	Window cai.Window `json:"window"`
	Origin string     `json:"origin"`
}

// feedMessageBQ format to persist in BQ
type feedMessageBQ struct {
	Asset  assetBQ    `json:"asset"`
	Window cai.Window `json:"window"`
	Origin string     `json:"origin"`
}

// asset Cloud Asset Metadata
type asset struct {
	Name                    string          `json:"name"`
	Owner                   string          `json:"owner"`
	ViolationResolver       string          `json:"violationResolver"`
	AncestryPathDisplayName string          `json:"ancestryPathDisplayName"`
	AncestryPath            string          `json:"ancestryPath"`
	AncestorsDisplayName    json.RawMessage `json:"ancestorsDisplayName"`
	Ancestors               json.RawMessage `json:"ancestors"`
	AssetType               string          `json:"assetType"`
	IamPolicy               json.RawMessage `json:"iamPolicy"`
	Resource                json.RawMessage `json:"resource"`
}

// assetBQ format to persist asset in BQ violations table
type assetBQ struct {
	Name                    string `json:"name"`
	Owner                   string `json:"owner"`
	ViolationResolver       string `json:"violationResolver"`
	AncestryPathDisplayName string `json:"ancestryPathDisplayName"`
	AncestryPath            string `json:"ancestryPath"`
	AncestorsDisplayName    string `json:"ancestorsDisplayName"`
	Ancestors               string `json:"ancestors"`
	AssetType               string `json:"assetType"`
	IamPolicy               string `json:"iamPolicy"`
	Resource                string `json:"resource"`
}

// assetFeedMessageBQ Cloud Asset Inventory feed message for asset table
type assetFeedMessageBQ struct {
	Asset   assetAssetBQ `json:"asset"`
	Window  cai.Window   `json:"window"`
	Deleted bool         `json:"deleted"`
	Origin  string       `json:"origin"`
}

// assetAssetBQ format to persist asset in BQ assets table
type assetAssetBQ struct {
	Name                    string    `json:"name"`
	Owner                   string    `json:"owner"`
	ViolationResolver       string    `json:"violationResolver"`
	AncestryPathDisplayName string    `json:"ancestryPathDisplayName"`
	AncestryPath            string    `json:"ancestryPath"`
	AncestorsDisplayName    []string  `json:"ancestorsDisplayName"`
	Ancestors               []string  `json:"ancestors"`
	AssetType               string    `json:"assetType"`
	Deleted                 bool      `json:"deleted"`
	Timestamp               time.Time `json:"timestamp"`
}

var originEventTimestamp time.Time

// Initialize is to be executed in the init() function of the cloud function to optimize the cold start
func Initialize(ctx context.Context, global *Global) (err error) {
	log.SetFlags(0)
	global.ctx = ctx

	var instanceDeployment InstanceDeployment
	var bigQueryClient *bigquery.Client
	var table *bigquery.Table

	initID := fmt.Sprintf("%v", uuid.New())
	err = ffo.ReadUnmarshalYAML(solution.PathToFunctionCode+solution.SettingsFileName, &instanceDeployment)
	if err != nil {
		log.Println(logging.Entry{
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

	log.Println(logging.Entry{
		MicroserviceName: global.microserviceName,
		InstanceName:     global.instanceName,
		Environment:      global.environment,
		Severity:         "NOTICE",
		Message:          "coldstart",
		InitID:           initID,
	})

	datasetName := instanceDeployment.Core.SolutionSettings.Hosting.Bigquery.Dataset.Name
	global.assetsCollectionID = instanceDeployment.Core.SolutionSettings.Hosting.FireStore.CollectionIDs.Assets
	global.ownerLabelKeyName = instanceDeployment.Core.SolutionSettings.Monitoring.LabelKeyNames.Owner
	global.retryTimeOutSeconds = instanceDeployment.Settings.Service.GCF.RetryTimeOutSeconds
	global.tableName = instanceDeployment.Settings.Instance.Bigquery.TableName
	global.violationResolverLabelKeyName = instanceDeployment.Core.SolutionSettings.Monitoring.LabelKeyNames.ViolationResolver
	projectID := instanceDeployment.Core.SolutionSettings.Hosting.ProjectID

	bigQueryClient, err = bigquery.NewClient(global.ctx, projectID)
	if err != nil {
		log.Println(logging.Entry{
			MicroserviceName: global.microserviceName,
			InstanceName:     global.instanceName,
			Environment:      global.environment,
			Severity:         "CRITICAL",
			Message:          "init_failed",
			Description:      fmt.Sprintf("bigquery.NewClient %v", err),
			InitID:           initID,
		})
		return err
	}
	dataset := bigQueryClient.Dataset(datasetName)
	_, err = dataset.Metadata(ctx)
	if err != nil {
		log.Println(logging.Entry{
			MicroserviceName: global.microserviceName,
			InstanceName:     global.instanceName,
			Environment:      global.environment,
			Severity:         "CRITICAL",
			Message:          "init_failed",
			Description:      fmt.Sprintf("dataset.Metadata %v", err),
			InitID:           initID,
		})
		return err
	}
	table = dataset.Table(global.tableName)
	_, err = table.Metadata(ctx)
	if err != nil {
		log.Println(logging.Entry{
			MicroserviceName: global.microserviceName,
			InstanceName:     global.instanceName,
			Environment:      global.environment,
			Severity:         "CRITICAL",
			Message:          "init_failed",
			Description:      fmt.Sprintf("missing table %s %v", global.tableName, err),
			InitID:           initID,
		})
		return err
	}
	global.inserter = table.Inserter()
	if global.tableName == "assets" {
		global.cloudresourcemanagerService, err = cloudresourcemanager.NewService(global.ctx)
		if err != nil {
			log.Println(logging.Entry{
				MicroserviceName: global.microserviceName,
				InstanceName:     global.instanceName,
				Environment:      global.environment,
				Severity:         "CRITICAL",
				Message:          "init_failed",
				Description:      fmt.Sprintf("cloudresourcemanager.NewService %v", err),
				InitID:           initID,
			})
			return err
		}
		global.cloudresourcemanagerServiceV2, err = cloudresourcemanagerv2.NewService(global.ctx)
		if err != nil {
			log.Println(logging.Entry{
				MicroserviceName: global.microserviceName,
				InstanceName:     global.instanceName,
				Environment:      global.environment,
				Severity:         "CRITICAL",
				Message:          "init_failed",
				Description:      fmt.Sprintf("cloudresourcemanagerv2.NewService %v", err),
				InitID:           initID,
			})
			return err
		}
		global.firestoreClient, err = firestore.NewClient(global.ctx, projectID)
		if err != nil {
			log.Println(logging.Entry{
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
	}
	return nil
}

// EntryPoint is the function to be executed for each cloud function occurence
func EntryPoint(ctxEvent context.Context, PubSubMessage gps.PubSubMessage, global *Global) error {
	// log.Println(string(PubSubMessage.Data))
	metadata, err := metadata.FromContext(ctxEvent)
	if err != nil {
		// Assume an error on the function invoker and try again.
		log.Println(logging.Entry{
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
	global.PubSubID = metadata.EventID
	now := time.Now()
	d := now.Sub(metadata.Timestamp)
	log.Println(logging.Entry{
		MicroserviceName:           global.microserviceName,
		InstanceName:               global.instanceName,
		Environment:                global.environment,
		Severity:                   "NOTICE",
		Message:                    "start",
		TriggeringPubsubID:         global.PubSubID,
		TriggeringPubsubAgeSeconds: d.Seconds(),
		TriggeringPubsubTimestamp:  metadata.Timestamp,
		Now:                        now,
	})

	if d.Seconds() > float64(global.retryTimeOutSeconds) {
		log.Println(logging.Entry{
			MicroserviceName:           global.microserviceName,
			InstanceName:               global.instanceName,
			Environment:                global.environment,
			Severity:                   "CRITICAL",
			Message:                    "noretry",
			Description:                "Pubsub message too old",
			TriggeringPubsubID:         global.PubSubID,
			TriggeringPubsubAgeSeconds: d.Seconds(),
			TriggeringPubsubTimestamp:  metadata.Timestamp,
			Now:                        now,
		})
		return nil
	}

	if strings.Contains(string(PubSubMessage.Data), "You have successfully configured real time feed") {
		log.Println(logging.Entry{
			MicroserviceName:   global.microserviceName,
			InstanceName:       global.instanceName,
			Environment:        global.environment,
			Severity:           "NOTICE",
			Message:            "cancel",
			Description:        fmt.Sprintf("ignored pubsub message: %s", string(PubSubMessage.Data)),
			TriggeringPubsubID: global.PubSubID,
		})
		return nil
	}
	var insertID string
	switch global.tableName {
	case "complianceStatus":
		insertID, err = persistComplianceStatus(PubSubMessage.Data, global)
	case "violations":
		insertID, err = persistViolation(PubSubMessage.Data, global)
	case "assets":
		insertID, err = persistAsset(PubSubMessage.Data, global)
	}
	if err != nil {
		log.Println(logging.Entry{
			MicroserviceName:   global.microserviceName,
			InstanceName:       global.instanceName,
			Environment:        global.environment,
			Severity:           "CRITICAL",
			Message:            "redo_on_transient",
			Description:        err.Error(),
			TriggeringPubsubID: global.PubSubID,
		})
		return err
	}
	if insertID != "" {
		now := time.Now()
		latency := now.Sub(metadata.Timestamp)
		latencyE2E := now.Sub(originEventTimestamp)
		step := logging.Step{
			StepID:        global.PubSubID,
			StepTimestamp: metadata.Timestamp,
		}
		var stepStack logging.Steps
		stepStack = append(stepStack, step)
		log.Println(logging.Entry{
			MicroserviceName:     global.microserviceName,
			InstanceName:         global.instanceName,
			Environment:          global.environment,
			Severity:             "NOTICE",
			Message:              "finish",
			Description:          fmt.Sprintf("insert %s ok %s", global.tableName, insertID),
			Now:                  now,
			TriggeringPubsubID:   global.PubSubID,
			OriginEventTimestamp: originEventTimestamp,
			LatencySeconds:       latency.Seconds(),
			LatencyE2ESeconds:    latencyE2E.Seconds(),
			StepStack:            stepStack,
		})
	}
	return nil
}

func persistComplianceStatus(pubSubJSONDoc []byte, global *Global) (insertID string, err error) {
	var complianceStatus monitor.ComplianceStatus
	err = json.Unmarshal(pubSubJSONDoc, &complianceStatus)
	if err != nil {
		log.Println(logging.Entry{
			MicroserviceName:   global.microserviceName,
			InstanceName:       global.instanceName,
			Environment:        global.environment,
			Severity:           "CRITICAL",
			Message:            "noretry",
			Description:        "json.Unmarshal(pubSubJSONDoc, &complianceStatus)",
			TriggeringPubsubID: global.PubSubID,
		})
		return "", nil
	}
	originEventTimestamp = complianceStatus.AssetInventoryTimeStamp

	insertID = fmt.Sprintf("%s%v%s%v", complianceStatus.AssetName, complianceStatus.AssetInventoryTimeStamp, complianceStatus.RuleName, complianceStatus.RuleDeploymentTimeStamp)
	savers := []*bigquery.StructSaver{
		{Struct: complianceStatus, Schema: gbq.GetComplianceStatusSchema(), InsertID: insertID},
	}
	if err := global.inserter.Put(global.ctx, savers); err != nil {
		return "", fmt.Errorf("inserter.Put %v %v", err, savers)
	}
	return insertID, nil
}

func persistViolation(pubSubJSONDoc []byte, global *Global) (insertID string, err error) {
	var violation violation
	var violationBQ violationBQ
	err = json.Unmarshal(pubSubJSONDoc, &violation)
	if err != nil {
		log.Println(logging.Entry{
			MicroserviceName:   global.microserviceName,
			InstanceName:       global.instanceName,
			Environment:        global.environment,
			Severity:           "CRITICAL",
			Message:            "noretry",
			Description:        "json.Unmarshal(pubSubJSONDoc, &violation)",
			TriggeringPubsubID: global.PubSubID,
		})
		return "", nil
	}
	originEventTimestamp = violation.FeedMessage.Window.StartTime

	violationBQ.NonCompliance.Message = violation.NonCompliance.Message
	violationBQ.NonCompliance.Metadata = string(violation.NonCompliance.Metadata)
	violationBQ.FunctionConfig = violation.FunctionConfig
	violationBQ.ConstraintConfig.Kind = violation.ConstraintConfig.Kind
	violationBQ.ConstraintConfig.Metadata.Name = violation.ConstraintConfig.Metadata.Name
	violationBQ.ConstraintConfig.Metadata.Annotation = string(violation.ConstraintConfig.Metadata.Annotation)
	violationBQ.ConstraintConfig.Spec.Severity = violation.ConstraintConfig.Spec.Severity
	violationBQ.ConstraintConfig.Spec.Match = string(violation.ConstraintConfig.Spec.Match)
	violationBQ.ConstraintConfig.Spec.Parameters = string(violation.ConstraintConfig.Spec.Parameters)
	violationBQ.FeedMessage.Window = violation.FeedMessage.Window
	violationBQ.FeedMessage.Origin = violation.FeedMessage.Origin
	violationBQ.FeedMessage.Asset.Name = violation.FeedMessage.Asset.Name
	violationBQ.FeedMessage.Asset.Owner = violation.FeedMessage.Asset.Owner
	violationBQ.FeedMessage.Asset.ViolationResolver = violation.FeedMessage.Asset.ViolationResolver
	violationBQ.FeedMessage.Asset.AssetType = violation.FeedMessage.Asset.AssetType
	violationBQ.FeedMessage.Asset.Ancestors = string(violation.FeedMessage.Asset.Ancestors)
	violationBQ.FeedMessage.Asset.AncestorsDisplayName = string(violation.FeedMessage.Asset.AncestorsDisplayName)
	violationBQ.FeedMessage.Asset.AncestryPath = violation.FeedMessage.Asset.AncestryPath
	violationBQ.FeedMessage.Asset.AncestryPathDisplayName = violation.FeedMessage.Asset.AncestryPathDisplayName
	violationBQ.FeedMessage.Asset.IamPolicy = string(violationBQ.FeedMessage.Asset.IamPolicy)
	violationBQ.FeedMessage.Asset.Resource = string(violation.FeedMessage.Asset.Resource)
	violationBQ.RegoModules = string(violation.RegoModules)

	insertID = fmt.Sprintf("%s%v%s%v%s", violationBQ.FeedMessage.Asset.Name, violation.FeedMessage.Window.StartTime, violation.FunctionConfig.FunctionName, violation.FunctionConfig.DeploymentTime, violation.NonCompliance.Message)
	savers := []*bigquery.StructSaver{
		{Struct: violationBQ, Schema: gbq.GetViolationsSchema(), InsertID: insertID},
	}
	if err := global.inserter.Put(global.ctx, savers); err != nil {
		return "", fmt.Errorf("inserter.Put %v", err)
	}
	return insertID, nil
}

func persistAsset(pubSubJSONDoc []byte, global *Global) (insertID string, err error) {
	var feedMessage feedMessage
	err = json.Unmarshal(pubSubJSONDoc, &feedMessage)
	if err != nil {
		log.Println(logging.Entry{
			MicroserviceName:   global.microserviceName,
			InstanceName:       global.instanceName,
			Environment:        global.environment,
			Severity:           "CRITICAL",
			Message:            "noretry",
			Description:        "json.Unmarshal(pubSubJSONDoc, &feedMessage)",
			TriggeringPubsubID: global.PubSubID,
		})
		return "", nil
	}
	var assetFeedMessageBQ assetFeedMessageBQ
	err = json.Unmarshal(pubSubJSONDoc, &assetFeedMessageBQ)
	if err != nil {
		log.Println(logging.Entry{
			MicroserviceName:   global.microserviceName,
			InstanceName:       global.instanceName,
			Environment:        global.environment,
			Severity:           "CRITICAL",
			Message:            "noretry",
			Description:        "json.Unmarshal(pubSubJSONDoc, &assetFeedMessageBQ)",
			TriggeringPubsubID: global.PubSubID,
		})
		return "", nil
	}
	if assetFeedMessageBQ.Asset.Name == "" {
		log.Println(logging.Entry{
			MicroserviceName:   global.microserviceName,
			InstanceName:       global.instanceName,
			Environment:        global.environment,
			Severity:           "CRITICAL",
			Message:            "noretry",
			Description:        "assetFeedMessageBQ.Asset.Name is empty",
			TriggeringPubsubID: global.PubSubID,
		})
		return "", nil
	}
	originEventTimestamp = feedMessage.Window.StartTime

	assetFeedMessageBQ.Asset.Timestamp = feedMessage.Window.StartTime
	assetFeedMessageBQ.Asset.Deleted = assetFeedMessageBQ.Deleted
	assetFeedMessageBQ.Asset.AncestryPath = cai.BuildAncestryPath(assetFeedMessageBQ.Asset.Ancestors)
	assetFeedMessageBQ.Asset.AncestorsDisplayName = cai.BuildAncestorsDisplayName(global.ctx, assetFeedMessageBQ.Asset.Ancestors, global.assetsCollectionID, global.firestoreClient, global.cloudresourcemanagerService, global.cloudresourcemanagerServiceV2)
	assetFeedMessageBQ.Asset.AncestryPathDisplayName = cai.BuildAncestryPath(assetFeedMessageBQ.Asset.AncestorsDisplayName)
	assetFeedMessageBQ.Asset.Owner, _ = cai.GetAssetLabelValue(global.ownerLabelKeyName, feedMessage.Asset.Resource)
	assetFeedMessageBQ.Asset.ViolationResolver, _ = cai.GetAssetLabelValue(global.violationResolverLabelKeyName, feedMessage.Asset.Resource)

	insertID = fmt.Sprintf("%s%v", assetFeedMessageBQ.Asset.Name, assetFeedMessageBQ.Asset.Timestamp)
	savers := []*bigquery.StructSaver{
		{Struct: assetFeedMessageBQ.Asset, Schema: gbq.GetAssetsSchema(), InsertID: insertID},
	}

	if err := global.inserter.Put(global.ctx, savers); err != nil {
		return "", fmt.Errorf("inserter.Put %v", err)
	}
	return insertID, nil
}
