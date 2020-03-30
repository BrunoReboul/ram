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

	"github.com/BrunoReboul/ram/utilities/ram"
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
	if global.retryTimeOutSeconds, ok = ram.GetEnvVarInt64("RETRYTIMEOUTSECONDS"); !ok {
		return
	}
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
		{Struct: complianceStatus, Schema: global.schema, InsertID: insertID},
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
		{Struct: violationBQ, Schema: global.schema, InsertID: insertID},
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
		{Struct: assetFeedMessageBQ.Asset, Schema: global.schema, InsertID: insertID},
	}

	if err := global.inserter.Put(global.ctx, savers); err != nil {
		return fmt.Errorf("inserter.Put %v", err)
	}
	log.Println("insert asset ok", insertID)
	return nil
}
