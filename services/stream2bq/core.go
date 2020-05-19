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

	"github.com/BrunoReboul/ram/utilities/ram"
	"google.golang.org/api/cloudresourcemanager/v1"

	"cloud.google.com/go/bigquery"
	"cloud.google.com/go/firestore"
	cloudresourcemanagerv2 "google.golang.org/api/cloudresourcemanager/v2"
)

// Global structure for global variables to optimize the cloud function performances
type Global struct {
	ctx                           context.Context
	initFailed                    bool
	retryTimeOutSeconds           int64
	assetsCollectionID            string
	cloudresourcemanagerService   *cloudresourcemanager.Service
	cloudresourcemanagerServiceV2 *cloudresourcemanagerv2.Service // v2 is needed for folders
	firestoreClient               *firestore.Client
	inserter                      *bigquery.Inserter
	ownerLabelKeyName             string
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
	Name        string          `json:"name"`
	Annotations json.RawMessage `json:"annotation"`
}

// constraintMetadataBQ format to persist in BQ
type constraintMetadataBQ struct {
	Name        string `json:"name"`
	Annotations string `json:"annotation"`
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
	Window ram.Window `json:"window"`
	Origin string     `json:"origin"`
}

// feedMessageBQ format to persist in BQ
type feedMessageBQ struct {
	Asset  assetBQ    `json:"asset"`
	Window ram.Window `json:"window"`
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
	Window  ram.Window   `json:"window"`
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
func Initialize(ctx context.Context, global *Global) {
	global.ctx = ctx
	global.initFailed = false

	// err is pre-declared to avoid shadowing client.
	var err error
	var instanceDeployment InstanceDeployment
	var bigQueryClient *bigquery.Client
	var dataset *bigquery.Dataset
	var table *bigquery.Table
	var tableNameList = []string{"complianceStatus", "violations", "assets"}

	log.Println("Function COLD START")
	err = ram.ReadUnmarshalYAML(fmt.Sprintf("./%s", ram.SettingsFileName), &instanceDeployment)
	if err != nil {
		log.Printf("ERROR - ReadUnmarshalYAML %s %v", ram.SettingsFileName, err)
		global.initFailed = true
		return
	}

	datasetLocation := instanceDeployment.Core.SolutionSettings.Hosting.Bigquery.Dataset.Location
	datasetName := instanceDeployment.Core.SolutionSettings.Hosting.Bigquery.Dataset.Name
	global.assetsCollectionID = instanceDeployment.Core.SolutionSettings.Hosting.FireStore.CollectionIDs.Assets
	global.ownerLabelKeyName = instanceDeployment.Core.SolutionSettings.Monitoring.LabelKeyNames.Owner
	global.retryTimeOutSeconds = instanceDeployment.Settings.Service.GCF.RetryTimeOutSeconds
	global.tableName = instanceDeployment.Settings.Instance.Bigquery.TableName
	global.violationResolverLabelKeyName = instanceDeployment.Core.SolutionSettings.Monitoring.LabelKeyNames.ViolationResolver
	projectID := instanceDeployment.Core.SolutionSettings.Hosting.ProjectID

	if !ram.Find(tableNameList, global.tableName) {
		log.Printf("ERROR - Unsupported tablename %s supported are %v\n", global.tableName, tableNameList)
		global.initFailed = true
		return
	}
	bigQueryClient, err = bigquery.NewClient(global.ctx, projectID)
	if err != nil {
		log.Printf("ERROR - bigquery.NewClient: %v", err)
		global.initFailed = true
		return
	}
	dataset, err = getDataset(global.ctx, datasetName, datasetLocation, bigQueryClient)
	if err != nil {
		log.Printf("ERROR - getDataset %s %v", datasetName, err)
		global.initFailed = true
		return
	}
	table, err = getTable(global.ctx, global.tableName, dataset)
	if err != nil {
		log.Printf("ERROR - getTable %s %v", global.tableName, err)
		global.initFailed = true
		return
	}
	global.inserter = table.Inserter()
	if global.tableName == "assets" {
		global.cloudresourcemanagerService, err = cloudresourcemanager.NewService(global.ctx)
		if err != nil {
			log.Printf("ERROR - cloudresourcemanager.NewService: %v", err)
			global.initFailed = true
			return
		}
		global.cloudresourcemanagerServiceV2, err = cloudresourcemanagerv2.NewService(global.ctx)
		if err != nil {
			log.Printf("ERROR - cloudresourcemanagerv2.NewService: %v", err)
			global.initFailed = true
			return
		}
		global.firestoreClient, err = firestore.NewClient(global.ctx, projectID)
		if err != nil {
			log.Printf("ERROR - firestore.NewClient: %v", err)
			global.initFailed = true
			return
		}
	}
}

func getDataset(ctx context.Context, datasetName string, location string, bigQueryClient *bigquery.Client) (dataset *bigquery.Dataset, err error) {
	dataset = bigQueryClient.Dataset(datasetName)
	datasetMetadata, err := dataset.Metadata(ctx)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "notfound") {
			var datasetToCreateMetadata bigquery.DatasetMetadata
			datasetToCreateMetadata.Name = datasetName
			datasetToCreateMetadata.Location = location
			datasetToCreateMetadata.Description = "Real-time Asset Monitor"
			datasetToCreateMetadata.Labels = map[string]string{"name": strings.ToLower(datasetName)}

			err = dataset.Create(ctx, &datasetToCreateMetadata)
			if err != nil {
				// deal with concurent executions
				if strings.Contains(strings.ToLower(err.Error()), "already exists") {
					datasetMetadata, err = dataset.Metadata(ctx)
					if err != nil {
						return nil, err
					}
				}
				return nil, fmt.Errorf("dataset.Create %v", err)
			}
			log.Printf("Created dataset %s", datasetName)
			return dataset, nil
		}
	}
	needToUpdate := false
	if datasetMetadata.Labels != nil {
		if value, ok := datasetMetadata.Labels["name"]; ok {
			if value != datasetMetadata.Name {
				needToUpdate = true
			}
		} else {
			needToUpdate = true
		}
	} else {
		needToUpdate = true
	}
	if needToUpdate {
		var datasetMetadataToUpdate bigquery.DatasetMetadataToUpdate
		datasetMetadataToUpdate.SetLabel("name", strings.ToLower(datasetName))
		datasetMetadata, err = dataset.Update(ctx, datasetMetadataToUpdate, "")
		if err != nil {
			return nil, fmt.Errorf("ERROR when updating dataset labels %v", err)
		}
		log.Printf("Update dataset labels %s", datasetName)
	}
	return dataset, nil
}

func getTable(ctx context.Context, tableName string, dataset *bigquery.Dataset) (table *bigquery.Table, err error) {
	var schema bigquery.Schema
	switch tableName {
	case "complianceStatus":
		schema = getComplianceStatusSchema()
	case "violations":
		schema = getViolationsSchema()
	case "assets":
		schema = getAssetsSchema()
	}

	table = dataset.Table(tableName)
	tableMetadata, err := table.Metadata(ctx)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "notfound") {
			var tableToCreateMetadata bigquery.TableMetadata
			tableToCreateMetadata.Name = tableName
			tableToCreateMetadata.Description = fmt.Sprintf("Real-time Asset Monitor - %s", tableName)
			tableToCreateMetadata.Labels = map[string]string{"name": strings.ToLower(tableName)}

			var timePartitioning bigquery.TimePartitioning
			timePartitioning.Expiration, _ = time.ParseDuration("24h")
			tableToCreateMetadata.TimePartitioning = &timePartitioning
			tableToCreateMetadata.Schema = schema

			err = table.Create(ctx, &tableToCreateMetadata)
			if err != nil {
				// deal with concurent executions
				if strings.Contains(strings.ToLower(err.Error()), "already exists") {
					tableMetadata, err = table.Metadata(ctx)
					if err != nil {
						return nil, err
					}
				}
				return nil, fmt.Errorf("table.Create %v", err)
			}
			log.Printf("Created table %s", tableName)
			return table, nil
		}
	}
	needToUpdate := false
	if tableMetadata.Labels != nil {
		if value, ok := tableMetadata.Labels["name"]; ok {
			if value != tableMetadata.Name {
				needToUpdate = true
			}
		} else {
			needToUpdate = true
		}
	} else {
		needToUpdate = true
	}
	if needToUpdate {
		var tableMetadataToUpdate bigquery.TableMetadataToUpdate
		tableMetadataToUpdate.SetLabel("name", strings.ToLower(tableName))
		tableMetadata, err = table.Update(ctx, tableMetadataToUpdate, "")
		if err != nil {
			return nil, fmt.Errorf("ERROR when updating table labels %v", err)
		}
		log.Printf("Update table labels %s", tableName)
	}
	return table, nil
}

// EntryPoint is the function to be executed for each cloud function occurence
func EntryPoint(ctxEvent context.Context, PubSubMessage ram.PubSubMessage, global *Global) error {
	// log.Println(string(PubSubMessage.Data))
	if ok, _, err := ram.IntialRetryCheck(ctxEvent, global.initFailed, global.retryTimeOutSeconds); !ok {
		return err
	}
	// log.Printf("EventType %s EventID %s Resource %s Timestamp %v", metadata.EventType, metadata.EventID, metadata.Resource.Type, metadata.Timestamp)

	var err error
	switch global.tableName {
	case "complianceStatus":
		err = persistComplianceStatus(PubSubMessage.Data, global)
	case "violations":
		err = persistViolation(PubSubMessage.Data, global)
	case "assets":
		err = persistAsset(PubSubMessage.Data, global)
	}
	if err != nil {
		return err // RETRY
	}
	return nil
}

func persistComplianceStatus(pubSubJSONDoc []byte, global *Global) error {
	var complianceStatus ram.ComplianceStatus
	err := json.Unmarshal(pubSubJSONDoc, &complianceStatus)
	if err != nil {
		log.Printf("ERROR - json.Unmarshal(pubSubJSONDoc, &complianceStatus): %v", err)
		return nil
	}
	insertID := fmt.Sprintf("%s%v%s%v", complianceStatus.AssetName, complianceStatus.AssetInventoryTimeStamp, complianceStatus.RuleName, complianceStatus.RuleDeploymentTimeStamp)
	savers := []*bigquery.StructSaver{
		{Struct: complianceStatus, Schema: getComplianceStatusSchema(), InsertID: insertID},
	}
	if err := global.inserter.Put(global.ctx, savers); err != nil {
		return fmt.Errorf("inserter.Put %v", err)
	}
	log.Println("insert complianceStatus ok", insertID)
	return nil
}

func persistViolation(pubSubJSONDoc []byte, global *Global) error {
	var violation violation
	var violationBQ violationBQ
	err := json.Unmarshal(pubSubJSONDoc, &violation)
	if err != nil {
		log.Printf("ERROR - json.Unmarshal(pubSubJSONDoc, &violation): %v", err)
		return nil
	}
	violationBQ.NonCompliance.Message = violation.NonCompliance.Message
	violationBQ.NonCompliance.Metadata = string(violation.NonCompliance.Metadata)
	violationBQ.FunctionConfig = violation.FunctionConfig
	violationBQ.ConstraintConfig.Kind = violation.ConstraintConfig.Kind
	violationBQ.ConstraintConfig.Metadata.Name = violation.ConstraintConfig.Metadata.Name
	violationBQ.ConstraintConfig.Metadata.Annotations = string(violation.ConstraintConfig.Metadata.Annotations)
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
		{Struct: violationBQ, Schema: getViolationsSchema(), InsertID: insertID},
	}
	if err := global.inserter.Put(global.ctx, savers); err != nil {
		return fmt.Errorf("inserter.Put %v", err)
	}
	log.Println("insert violation ok", insertID)
	return nil
}

func persistAsset(pubSubJSONDoc []byte, global *Global) error {
	var feedMessage feedMessage
	err := json.Unmarshal(pubSubJSONDoc, &feedMessage)
	if err != nil {
		log.Printf("ERROR - json.Unmarshal(pubSubJSONDoc, &feedMessage): %v", err)
		return nil
	}
	var assetFeedMessageBQ assetFeedMessageBQ
	err = json.Unmarshal(pubSubJSONDoc, &assetFeedMessageBQ)
	if err != nil {
		log.Printf("ERROR - json.Unmarshal(pubSubJSONDoc, &assetFeedMessageBQ): %v", err)
		return nil
	}
	if assetFeedMessageBQ.Asset.Name == "" {
		log.Printf("ERROR - assetFeedMessageBQ.Asset.Name is empty")
		return nil
	}
	assetFeedMessageBQ.Asset.Timestamp = feedMessage.Window.StartTime
	assetFeedMessageBQ.Asset.Deleted = assetFeedMessageBQ.Deleted
	assetFeedMessageBQ.Asset.AncestryPath = ram.BuildAncestryPath(assetFeedMessageBQ.Asset.Ancestors)
	assetFeedMessageBQ.Asset.AncestorsDisplayName = ram.BuildAncestorsDisplayName(global.ctx, assetFeedMessageBQ.Asset.Ancestors, global.assetsCollectionID, global.firestoreClient, global.cloudresourcemanagerService, global.cloudresourcemanagerServiceV2)
	assetFeedMessageBQ.Asset.AncestryPathDisplayName = ram.BuildAncestryPath(assetFeedMessageBQ.Asset.AncestorsDisplayName)
	assetFeedMessageBQ.Asset.Owner, _ = ram.GetAssetContact(global.ownerLabelKeyName, feedMessage.Asset.Resource)
	assetFeedMessageBQ.Asset.ViolationResolver, _ = ram.GetAssetContact(global.violationResolverLabelKeyName, feedMessage.Asset.Resource)

	insertID := fmt.Sprintf("%s%v", assetFeedMessageBQ.Asset.Name, assetFeedMessageBQ.Asset.Timestamp)
	savers := []*bigquery.StructSaver{
		{Struct: assetFeedMessageBQ.Asset, Schema: getAssetsSchema(), InsertID: insertID},
	}

	if err := global.inserter.Put(global.ctx, savers); err != nil {
		return fmt.Errorf("inserter.Put %v", err)
	}
	log.Println("insert asset ok", insertID)
	return nil
}

func getViolationsSchema() bigquery.Schema {
	return bigquery.Schema{
		{
			Name:        "nonCompliance",
			Type:        bigquery.RecordFieldType,
			Description: "The violation information, aka why it is not compliant",
			Schema: bigquery.Schema{
				{Name: "message", Required: true, Type: bigquery.StringFieldType},
				{Name: "metadata", Required: false, Type: bigquery.StringFieldType},
			},
		},
		{
			Name:        "functionConfig",
			Type:        bigquery.RecordFieldType,
			Description: "The settings of the cloud function hosting the rule check",
			Schema: bigquery.Schema{
				{Name: "functionName", Required: true, Type: bigquery.StringFieldType},
				{Name: "deploymentTime", Required: true, Type: bigquery.TimestampFieldType},
				{Name: "projectID", Required: false, Type: bigquery.StringFieldType},
				{Name: "environment", Required: false, Type: bigquery.StringFieldType},
			},
		},
		{
			Name:        "constraintConfig",
			Type:        bigquery.RecordFieldType,
			Description: "The settings of the constraint used in conjonction with the rego template to assess the rule",
			Schema: bigquery.Schema{
				{Name: "kind", Required: false, Type: bigquery.StringFieldType},
				{
					Name: "metadata",
					Type: bigquery.RecordFieldType,
					Schema: bigquery.Schema{
						{Name: "name", Required: false, Type: bigquery.StringFieldType},
						{Name: "annotation", Required: false, Type: bigquery.StringFieldType},
					},
				},
				{
					Name: "spec",
					Type: bigquery.RecordFieldType,
					Schema: bigquery.Schema{
						{Name: "severity", Required: false, Type: bigquery.StringFieldType},
						{Name: "match", Required: false, Type: bigquery.StringFieldType},
						{Name: "parameters", Required: false, Type: bigquery.StringFieldType},
					},
				},
			},
		},
		{
			Name:        "feedMessage",
			Type:        bigquery.RecordFieldType,
			Description: "The message from Cloud Asset Inventory in realtime or from split dump in batch",
			Schema: bigquery.Schema{
				{
					Name: "asset",
					Type: bigquery.RecordFieldType,
					Schema: bigquery.Schema{
						{Name: "name", Required: true, Type: bigquery.StringFieldType},
						{Name: "owner", Required: false, Type: bigquery.StringFieldType},
						{Name: "violationResolver", Required: false, Type: bigquery.StringFieldType},
						{Name: "ancestryPathDisplayName", Required: false, Type: bigquery.StringFieldType},
						{Name: "ancestryPath", Required: false, Type: bigquery.StringFieldType},
						{Name: "ancestorsDisplayName", Required: false, Type: bigquery.StringFieldType},
						{Name: "ancestors", Required: false, Type: bigquery.StringFieldType},
						{Name: "assetType", Required: true, Type: bigquery.StringFieldType},
						{Name: "iamPolicy", Required: false, Type: bigquery.StringFieldType},
						{Name: "resource", Required: false, Type: bigquery.StringFieldType},
					},
				},
				{
					Name: "window",
					Type: bigquery.RecordFieldType,
					Schema: bigquery.Schema{
						{Name: "startTime", Required: true, Type: bigquery.TimestampFieldType},
					},
				},
				{Name: "origin", Required: false, Type: bigquery.StringFieldType},
			},
		},
		{Name: "regoModules", Required: false, Type: bigquery.StringFieldType, Description: "The rego code, including the rule template used to assess the rule as a JSON document"},
	}
}

func getComplianceStatusSchema() bigquery.Schema {
	return bigquery.Schema{
		{Name: "assetName", Required: true, Type: bigquery.StringFieldType},
		{Name: "assetInventoryTimeStamp", Required: true, Type: bigquery.TimestampFieldType, Description: "When the asset change was captured"},
		{Name: "assetInventoryOrigin", Required: false, Type: bigquery.StringFieldType, Description: "Mean to capture the asset change: real-time or batch-export"},
		{Name: "ruleName", Required: true, Type: bigquery.StringFieldType},
		{Name: "ruleDeploymentTimeStamp", Required: true, Type: bigquery.TimestampFieldType, Description: "When the rule was assessed"},
		{Name: "compliant", Required: true, Type: bigquery.BooleanFieldType},
		{Name: "deleted", Required: true, Type: bigquery.BooleanFieldType},
	}
}

func getAssetsSchema() bigquery.Schema {
	return bigquery.Schema{
		{Name: "timestamp", Required: true, Type: bigquery.TimestampFieldType},
		{Name: "name", Required: true, Type: bigquery.StringFieldType},
		{Name: "owner", Required: false, Type: bigquery.StringFieldType},
		{Name: "violationResolver", Required: false, Type: bigquery.StringFieldType},
		{Name: "ancestryPathDisplayName", Required: false, Type: bigquery.StringFieldType},
		{Name: "ancestryPath", Required: false, Type: bigquery.StringFieldType},
		{Name: "ancestorsDisplayName", Repeated: true, Type: bigquery.StringFieldType},
		{Name: "ancestors", Repeated: true, Type: bigquery.StringFieldType},
		{Name: "assetType", Required: true, Type: bigquery.TimestampFieldType},
		{Name: "deleted", Required: true, Type: bigquery.BooleanFieldType},
	}
}
