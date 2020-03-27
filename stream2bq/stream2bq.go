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
	"io/ioutil"
	"log"
	"os"
	"time"

	"github.com/BrunoReboul/ram/helper"
	"google.golang.org/api/cloudresourcemanager/v1"

	"cloud.google.com/go/bigquery"
	"cloud.google.com/go/firestore"
	cloudresourcemanagerv2 "google.golang.org/api/cloudresourcemanager/v2"
)

// Global structure for global variables to optimize the cloud function performances
type Global struct {
	ctx                           context.Context
	assetsCollectionID            string
	cloudresourcemanagerService   *cloudresourcemanager.Service
	cloudresourcemanagerServiceV2 *cloudresourcemanagerv2.Service // v2 is needed for folders
	firestoreClient               *firestore.Client
	initFailed                    bool
	inserter                      *bigquery.Inserter
	ownerLabelKeyName             string
	retryTimeOutSeconds           int64
	schema                        bigquery.Schema
	tableName                     string
	violationResolverLabelKeyName string
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

// Violation from the "audit" rego policy in "audit.rego" module
type Violation struct {
	NonCompliance    NonCompliance    `json:"nonCompliance"`
	FunctionConfig   FunctionConfig   `json:"functionConfig"`
	ConstraintConfig ConstraintConfig `json:"constraintConfig"`
	FeedMessage      FeedMessage      `json:"feedMessage"`
	RegoModules      json.RawMessage  `json:"regoModules"`
}

// ViolationBQ from the "audit" rego policy in "audit.rego" module
type ViolationBQ struct {
	NonCompliance    NonComplianceBQ    `json:"nonCompliance"`
	FunctionConfig   FunctionConfig     `json:"functionConfig"`
	ConstraintConfig ConstraintConfigBQ `json:"constraintConfig"`
	FeedMessage      FeedMessageBQ      `json:"feedMessage"`
	RegoModules      string             `json:"regoModules"`
}

// NonCompliance form the "deny" rego policy in a <templateName>.rego module
type NonCompliance struct {
	Message  string          `json:"message"`
	Metadata json.RawMessage `json:"metadata"`
}

// NonComplianceBQ form the "deny" rego policy in a <templateName>.rego module
type NonComplianceBQ struct {
	Message  string `json:"message"`
	Metadata string `json:"metadata"`
}

// FunctionConfig function deployment settings
type FunctionConfig struct {
	FunctionName   string    `json:"functionName"`
	DeploymentTime time.Time `json:"deploymentTime"`
	ProjectID      string    `json:"projectID"`
	Environment    string    `json:"environment"`
}

// ConstraintConfig expose content of the constraint yaml file
type ConstraintConfig struct {
	APIVersion string             `json:"apiVersion"`
	Kind       string             `json:"kind"`
	Metadata   ConstraintMetadata `json:"metadata"`
	Spec       Spec               `json:"spec"`
}

// ConstraintConfigBQ format to persist in BQ
type ConstraintConfigBQ struct {
	Kind     string               `json:"kind"`
	Metadata ConstraintMetadataBQ `json:"metadata"`
	Spec     SpecBQ               `json:"spec"`
}

// ConstraintMetadata Constraint's metadata
type ConstraintMetadata struct {
	Name        string          `json:"name"`
	Annotations json.RawMessage `json:"annotation"`
}

// ConstraintMetadataBQ format to persist in BQ
type ConstraintMetadataBQ struct {
	Name        string `json:"name"`
	Annotations string `json:"annotation"`
}

// Spec Constraint's specifications
type Spec struct {
	Severity   string          `json:"severity"`
	Match      json.RawMessage `json:"match"`
	Parameters json.RawMessage `json:"parameters"`
}

// SpecBQ format to persist in BQ
type SpecBQ struct {
	Severity   string `json:"severity"`
	Match      string `json:"match"`
	Parameters string `json:"parameters"`
}

// FeedMessage Cloud Asset Inventory feed message
type FeedMessage struct {
	Asset  Asset         `json:"asset"`
	Window helper.Window `json:"window"`
	Origin string        `json:"origin"`
}

// FeedMessageBQ format to persist in BQ
type FeedMessageBQ struct {
	Asset  AssetBQ       `json:"asset"`
	Window helper.Window `json:"window"`
	Origin string        `json:"origin"`
}

// Asset Cloud Asset Metadata
type Asset struct {
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

// AssetBQ format to persist asset in BQ violations table
type AssetBQ struct {
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

// AssetFeedMessageBQ Cloud Asset Inventory feed message for asset table
type AssetFeedMessageBQ struct {
	Asset   AssetAssetBQ  `json:"asset"`
	Window  helper.Window `json:"window"`
	Deleted bool          `json:"deleted"`
	Origin  string        `json:"origin"`
}

// AssetAssetBQ format to persist asset in BQ assets table
type AssetAssetBQ struct {
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

// Parameters Constraint's settings
type Parameters map[string]json.RawMessage

// Initialize is to be executed in the init() function of the cloud function to optimize the cold start
func Initialize(ctx context.Context, global *Global) {
	global.ctx = ctx
	global.initFailed = false

	var bigQueryClient *bigquery.Client
	var dataset *bigquery.Dataset
	var datasetName string
	var ok bool
	var projectID string
	var schemaFileName string
	var table *bigquery.Table
	var tableNameList = []string{"complianceStatus", "violations", "assets"}

	datasetName = os.Getenv("BQ_DATASET")
	global.assetsCollectionID = os.Getenv("ASSETSCOLLECTIONID")
	global.ownerLabelKeyName = os.Getenv("OWNERLABELKEYNAME")
	global.tableName = os.Getenv("BQ_TABLE")
	global.violationResolverLabelKeyName = os.Getenv("VIOLATIONRESOLVERLABELKEYNAME")
	projectID = os.Getenv("GCP_PROJECT")
	schemaFileName = "./schema.json"

	log.Println("Function COLD START")
	// err is pre-declared to avoid shadowing client.
	var err error
	if global.retryTimeOutSeconds, ok = helper.GetEnvVarInt64("RETRYTIMEOUTSECONDS"); !ok {
		return
	}
	if !helper.Find(tableNameList, global.tableName) {
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
	dataset = bigQueryClient.Dataset(datasetName)
	_, err = dataset.Metadata(ctx)
	if err != nil {
		log.Printf("ERROR - dataset.Metadata: %v", err)
		global.initFailed = true
		return
	}
	table = dataset.Table(global.tableName)
	_, err = table.Metadata(ctx)
	if err != nil {
		log.Printf("ERROR - missing table %s %v", global.tableName, err)
		global.initFailed = true
		return
	}
	global.inserter = table.Inserter()
	schemaFileContent, err := ioutil.ReadFile(schemaFileName)
	if err != nil {
		log.Printf("ERROR - ioutil.ReadFile: %v", err)
		global.initFailed = true
		return
	}
	global.schema, err = bigquery.SchemaFromJSON(schemaFileContent)
	if err != nil {
		log.Printf("ERROR - bigquery.SchemaFromJSON: %v", err)
		global.initFailed = true
		return
	}
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

// EntryPoint is the function to be executed for each cloud function occurence
func EntryPoint(ctxEvent context.Context, PubSubMessage helper.PubSubMessage, global *Global) error {
	// log.Println(string(PubSubMessage.Data))
	if ok, _, err := helper.IntialRetryCheck(ctxEvent, global.initFailed, global.retryTimeOutSeconds); !ok {
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
	var complianceStatus ComplianceStatus
	err := json.Unmarshal(pubSubJSONDoc, &complianceStatus)
	if err != nil {
		log.Printf("ERROR - json.Unmarshal(pubSubJSONDoc, &complianceStatus): %v", err)
		return nil
	}
	insertID := fmt.Sprintf("%s%v%s%v", complianceStatus.AssetName, complianceStatus.AssetInventoryTimeStamp, complianceStatus.RuleName, complianceStatus.RuleDeploymentTimeStamp)
	savers := []*bigquery.StructSaver{
		{Struct: complianceStatus, Schema: global.schema, InsertID: insertID},
	}
	if err := global.inserter.Put(global.ctx, savers); err != nil {
		return fmt.Errorf("inserter.Put %v", err)
	}
	log.Println("insert complianceStatus ok", insertID)
	return nil
}

func persistViolation(pubSubJSONDoc []byte, global *Global) error {
	var violation Violation
	var violationBQ ViolationBQ
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
		{Struct: violationBQ, Schema: global.schema, InsertID: insertID},
	}
	if err := global.inserter.Put(global.ctx, savers); err != nil {
		return fmt.Errorf("inserter.Put %v", err)
	}
	log.Println("insert violation ok", insertID)
	return nil
}

func persistAsset(pubSubJSONDoc []byte, global *Global) error {
	var feedMessage FeedMessage
	err := json.Unmarshal(pubSubJSONDoc, &feedMessage)
	if err != nil {
		log.Printf("ERROR - json.Unmarshal(pubSubJSONDoc, &feedMessage): %v", err)
		return nil
	}
	var assetFeedMessageBQ AssetFeedMessageBQ
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
	assetFeedMessageBQ.Asset.AncestryPath = helper.BuildAncestryPath(assetFeedMessageBQ.Asset.Ancestors)
	assetFeedMessageBQ.Asset.AncestorsDisplayName = helper.BuildAncestorsDisplayName(global.ctx, assetFeedMessageBQ.Asset.Ancestors, global.assetsCollectionID, global.firestoreClient, global.cloudresourcemanagerService, global.cloudresourcemanagerServiceV2)
	assetFeedMessageBQ.Asset.AncestryPathDisplayName = helper.BuildAncestryPath(assetFeedMessageBQ.Asset.AncestorsDisplayName)
	assetFeedMessageBQ.Asset.Owner, _ = helper.GetAssetContact(global.ownerLabelKeyName, feedMessage.Asset.Resource)
	assetFeedMessageBQ.Asset.ViolationResolver, _ = helper.GetAssetContact(global.violationResolverLabelKeyName, feedMessage.Asset.Resource)

	insertID := fmt.Sprintf("%s%v", assetFeedMessageBQ.Asset.Name, assetFeedMessageBQ.Asset.Timestamp)
	savers := []*bigquery.StructSaver{
		{Struct: assetFeedMessageBQ.Asset, Schema: global.schema, InsertID: insertID},
	}

	if err := global.inserter.Put(global.ctx, savers); err != nil {
		return fmt.Errorf("inserter.Put %v", err)
	}
	log.Println("insert asset ok", insertID)
	return nil
}
