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
	firestoreClient               *firestore.Client
	inserter                      *bigquery.Inserter
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

// Initialize is to be executed in the init() function of the cloud function to optimize the cold start
func Initialize(ctx context.Context, global *Global) (err error) {
	log.SetFlags(0)
	global.ctx = ctx

	var instanceDeployment InstanceDeployment
	var bigQueryClient *bigquery.Client
	var table *bigquery.Table

	logEntryPrefix := fmt.Sprintf("init_id %s", uuid.New())
	log.Printf("%s function COLD START", logEntryPrefix)
	err = ffo.ReadUnmarshalYAML(solution.PathToFunctionCode+solution.SettingsFileName, &instanceDeployment)
	if err != nil {
		return fmt.Errorf("%s ReadUnmarshalYAML %s %v", logEntryPrefix, solution.SettingsFileName, err)
	}

	datasetName := instanceDeployment.Core.SolutionSettings.Hosting.Bigquery.Dataset.Name
	global.assetsCollectionID = instanceDeployment.Core.SolutionSettings.Hosting.FireStore.CollectionIDs.Assets
	global.ownerLabelKeyName = instanceDeployment.Core.SolutionSettings.Monitoring.LabelKeyNames.Owner
	global.retryTimeOutSeconds = instanceDeployment.Settings.Service.GCF.RetryTimeOutSeconds
	global.tableName = instanceDeployment.Settings.Instance.Bigquery.TableName
	global.violationResolverLabelKeyName = instanceDeployment.Core.SolutionSettings.Monitoring.LabelKeyNames.ViolationResolver
	projectID := instanceDeployment.Core.SolutionSettings.Hosting.ProjectID

	bigQueryClient, err = bigquery.NewClient(global.ctx, projectID)
	if err != nil {
		return fmt.Errorf("%s bigquery.NewClient: %v", logEntryPrefix, err)
	}
	dataset := bigQueryClient.Dataset(datasetName)
	_, err = dataset.Metadata(ctx)
	if err != nil {
		return fmt.Errorf("%s dataset.Metadata: %v", logEntryPrefix, err)
	}
	table = dataset.Table(global.tableName)
	_, err = table.Metadata(ctx)
	if err != nil {
		return fmt.Errorf("%s missing table %s %v", logEntryPrefix, global.tableName, err)
	}
	global.inserter = table.Inserter()
	if global.tableName == "assets" {
		global.cloudresourcemanagerService, err = cloudresourcemanager.NewService(global.ctx)
		if err != nil {
			return fmt.Errorf("%s cloudresourcemanager.NewService: %v", logEntryPrefix, err)
		}
		global.cloudresourcemanagerServiceV2, err = cloudresourcemanagerv2.NewService(global.ctx)
		if err != nil {
			return fmt.Errorf("%s cloudresourcemanagerv2.NewService: %v", logEntryPrefix, err)
		}
		global.firestoreClient, err = firestore.NewClient(global.ctx, projectID)
		if err != nil {
			return fmt.Errorf("%s firestore.NewClient: %v", logEntryPrefix, err)
		}
	}
	return nil
}

// EntryPoint is the function to be executed for each cloud function occurence
func EntryPoint(ctxEvent context.Context, PubSubMessage gps.PubSubMessage, global *Global) error {
	// log.Println(string(PubSubMessage.Data))
	var logEntry logging.Entry
	metadata, err := metadata.FromContext(ctxEvent)
	if err != nil {
		// Assume an error on the function invoker and try again.
		return fmt.Errorf("pubsub_id no available REDO_ON_TRANSIENT metadata.FromContext: %v", err)
	}
	global.PubSubID = metadata.EventID

	now := time.Now()
	d := now.Sub(metadata.Timestamp)
	log.Printf("pubsub_id %s age sec %v now %v event timestamp %s", global.PubSubID, d.Seconds(), now, metadata.Timestamp)

	logEntry.Severity = "INFO"
	logEntry.Message = fmt.Sprintf("pubsub_id %s age sec %v now %v event timestamp %s", global.PubSubID, d.Seconds(), now, metadata.Timestamp)
	logEntry.Component = global.PubSubID

	logEntry.Message = "fmt.Printf(logEntry.String())" + logEntry.Message
	fmt.Printf(logEntry.String())
	logEntry.Message = "log.Println(logEntry.String())" + logEntry.Message
	log.Println(logEntry.String())

	log.Println(logging.Entry{
		Severity:  "INFO",
		Message:   "log.Println(logging.Entry{" + fmt.Sprintf("pubsub_id %s age sec %v now %v event timestamp %s", global.PubSubID, d.Seconds(), now, metadata.Timestamp),
		Component: global.PubSubID,
	})

	if d.Seconds() > float64(global.retryTimeOutSeconds) {
		log.Printf("pubsub_id %s NORETRY_ERROR pubsub message too old. max age sec %d now %v event timestamp %s", global.PubSubID, global.retryTimeOutSeconds, now, metadata.Timestamp)
		return nil
	}

	if strings.Contains(string(PubSubMessage.Data), "You have successfully configured real time feed") {
		log.Printf("pubsub_id %s ignored pubsub message: %s", global.PubSubID, string(PubSubMessage.Data))
		return nil
	}
	switch global.tableName {
	case "complianceStatus":
		err = persistComplianceStatus(PubSubMessage.Data, global)
	case "violations":
		err = persistViolation(PubSubMessage.Data, global)
	case "assets":
		err = persistAsset(PubSubMessage.Data, global)
	}
	if err != nil {
		return fmt.Errorf("pubsub_id %s REDO_ON_TRANSIENT %v", global.PubSubID, err)
	}
	// log.Printf("pubsub_id %s exit nil", global.PubSubID)
	return nil
}

func persistComplianceStatus(pubSubJSONDoc []byte, global *Global) error {
	var complianceStatus monitor.ComplianceStatus
	err := json.Unmarshal(pubSubJSONDoc, &complianceStatus)
	if err != nil {
		log.Printf("pubsub_id %s NORETRY_ERROR json.Unmarshal(pubSubJSONDoc, &complianceStatus) %s %v", global.PubSubID, string(pubSubJSONDoc), err)
		return nil
	}
	insertID := fmt.Sprintf("%s%v%s%v", complianceStatus.AssetName, complianceStatus.AssetInventoryTimeStamp, complianceStatus.RuleName, complianceStatus.RuleDeploymentTimeStamp)
	savers := []*bigquery.StructSaver{
		{Struct: complianceStatus, Schema: gbq.GetComplianceStatusSchema(), InsertID: insertID},
	}
	if err := global.inserter.Put(global.ctx, savers); err != nil {
		return fmt.Errorf("inserter.Put %v %v", err, savers)
	}
	log.Printf("pubsub_id %s insert complianceStatus ok %s", global.PubSubID, insertID)
	return nil
}

func persistViolation(pubSubJSONDoc []byte, global *Global) error {
	var violation violation
	var violationBQ violationBQ
	err := json.Unmarshal(pubSubJSONDoc, &violation)
	if err != nil {
		log.Printf("pubsub_id %s NORETRY_ERROR json.Unmarshal(pubSubJSONDoc, &violation): %s %v", global.PubSubID, string(pubSubJSONDoc), err)
		return nil
	}
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

	// violationBQJSON, err := json.Marshal(violationBQ)
	// if err != nil {
	// 	log.Fatalln(err)
	// }
	// log.Println(string(violationBQJSON))

	insertID := fmt.Sprintf("%s%v%s%v%s", violationBQ.FeedMessage.Asset.Name, violation.FeedMessage.Window.StartTime, violation.FunctionConfig.FunctionName, violation.FunctionConfig.DeploymentTime, violation.NonCompliance.Message)
	savers := []*bigquery.StructSaver{
		{Struct: violationBQ, Schema: gbq.GetViolationsSchema(), InsertID: insertID},
	}
	if err := global.inserter.Put(global.ctx, savers); err != nil {
		return fmt.Errorf("inserter.Put %v", err)
	}
	log.Printf("pubsub_id %s insert violation ok %s", global.PubSubID, insertID)
	return nil
}

func persistAsset(pubSubJSONDoc []byte, global *Global) error {
	var feedMessage feedMessage
	err := json.Unmarshal(pubSubJSONDoc, &feedMessage)
	if err != nil {
		log.Printf("pubsub_id %s NORETRY_ERROR json.Unmarshal(pubSubJSONDoc, &feedMessage) %s %v", global.PubSubID, string(pubSubJSONDoc), err)
		return nil
	}
	var assetFeedMessageBQ assetFeedMessageBQ
	err = json.Unmarshal(pubSubJSONDoc, &assetFeedMessageBQ)
	if err != nil {
		log.Printf("pubsub_id %s NORETRY_ERROR json.Unmarshal(pubSubJSONDoc, &assetFeedMessageBQ): %v", global.PubSubID, err)
		return nil
	}
	if assetFeedMessageBQ.Asset.Name == "" {
		log.Printf("pubsub_id %s NORETRY_ERROR assetFeedMessageBQ.Asset.Name is empty", global.PubSubID)
		return nil
	}
	assetFeedMessageBQ.Asset.Timestamp = feedMessage.Window.StartTime
	assetFeedMessageBQ.Asset.Deleted = assetFeedMessageBQ.Deleted
	assetFeedMessageBQ.Asset.AncestryPath = cai.BuildAncestryPath(assetFeedMessageBQ.Asset.Ancestors)
	assetFeedMessageBQ.Asset.AncestorsDisplayName = cai.BuildAncestorsDisplayName(global.ctx, assetFeedMessageBQ.Asset.Ancestors, global.assetsCollectionID, global.firestoreClient, global.cloudresourcemanagerService, global.cloudresourcemanagerServiceV2)
	assetFeedMessageBQ.Asset.AncestryPathDisplayName = cai.BuildAncestryPath(assetFeedMessageBQ.Asset.AncestorsDisplayName)
	assetFeedMessageBQ.Asset.Owner, _ = cai.GetAssetLabelValue(global.ownerLabelKeyName, feedMessage.Asset.Resource)
	assetFeedMessageBQ.Asset.ViolationResolver, _ = cai.GetAssetLabelValue(global.violationResolverLabelKeyName, feedMessage.Asset.Resource)

	insertID := fmt.Sprintf("%s%v", assetFeedMessageBQ.Asset.Name, assetFeedMessageBQ.Asset.Timestamp)
	savers := []*bigquery.StructSaver{
		{Struct: assetFeedMessageBQ.Asset, Schema: gbq.GetAssetsSchema(), InsertID: insertID},
	}

	if err := global.inserter.Put(global.ctx, savers); err != nil {
		return fmt.Errorf("inserter.Put %v", err)
	}
	log.Printf("pubsub_id %s insert asset ok %s", global.PubSubID, insertID)
	return nil
}
