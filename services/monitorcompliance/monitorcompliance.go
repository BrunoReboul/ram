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

// Package monitorcompliance check asset compliance. Is the heart of RAM
// - Triggered by: resource or IAM policies assets feed messages in PubSub topics
// - Instances:
//   - one per REGO rule
//   - all constraints (yaml settings) related to a REGO rule are evaluated in the REGO rule instance
// - Output:
//   - PubSub violation topic
//   - PubSub complianceStatus topic
// - Cardinality:
//   - When compliant one-one only the compliance state, no violations
//   - When not compliant one-few 1 compliance state + n violations
// - Automatic retrying: yes
// - Required environment variables:
//   - ASSETSCOLLECTIONID the name of the FireStore collection grouping all assets documents
//   - ENVIRONMENT the execution environment for RAM, eg, dev
//   - OWNERLABELKEYNAME key name for the label identifying the asset owner
//   - STATUS_TOPIC name of the PubSub topic used to output evaluated compliance states
//   - VIOLATIONRESOLVERLABELKEYNAMEkey name for the label identifying the asset violation resolver
//   - VIOLATION_TOPIC name of the PubSub topic used to output found violations
package monitorcompliance

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"time"

	"cloud.google.com/go/firestore"
	"cloud.google.com/go/pubsub"
	"github.com/BrunoReboul/ram/utilities/ram"
	"github.com/open-policy-agent/opa/rego"
	"google.golang.org/api/cloudresourcemanager/v1"
	cloudresourcemanagerv2 "google.golang.org/api/cloudresourcemanager/v2"
)

const settingsFileName string = "./settings.json"

// Global structure for global variables to optimize the cloud function performances
type Global struct {
	ctx                           context.Context
	initFailed                    bool
	retryTimeOutSeconds           int64
	assetsCollectionID            string
	assetsFilePath                string
	assetsFolderPath              string
	cloudresourcemanagerService   *cloudresourcemanager.Service
	cloudresourcemanagerServiceV2 *cloudresourcemanagerv2.Service // v2 is needed for folders
	deploymentTime                time.Time
	environment                   string
	firestoreClient               *firestore.Client
	functionName                  string
	ownerLabelKeyName             string
	projectID                     string
	pubSubClient                  *pubsub.Client
	ramComplianceStatusTopicName  string
	ramViolationTopicName         string
	regoModulesFolderPath         string
	settings                      Settings
	violationResolverLabelKeyName string
}

// Settings the structure of the settings.json setting file
type Settings struct {
	WritabelOPAFolderPath string `json:"writabelOPAFolderPath"`
	AssetsFolderName      string `json:"assetsFolderName"`
	AssetsFileName        string `json:"assetsFileName"`
	OPAFolderPath         string `json:"OPAFolderPath"`
	RegoModulesFolderName string `json:"regoModulesFolderName"`
}

// FeedMessage Cloud Asset Inventory feed message
type FeedMessage struct {
	Asset   Asset      `json:"asset"`
	Window  ram.Window `json:"window"`
	Deleted bool       `json:"deleted"`
	Origin  string     `json:"origin"`
}

// Asset Cloud Asset Metadata
// Duplicate "iamPolicy" and "assetType en ensure compatibility beetween format in CAI feed, aka real time, and CAI Export aka batch
type Asset struct {
	Name                    string          `json:"name"`
	Owner                   string          `json:"owner"`
	ViolationResolver       string          `json:"violationResolver"`
	AncestryPathDisplayName string          `json:"ancestryPathDisplayName"`
	AncestryPath            string          `json:"ancestryPath"`
	AncestryPathLegacy      string          `json:"ancestry_path"`
	AncestorsDisplayName    []string        `json:"ancestorsDisplayName"`
	Ancestors               []string        `json:"ancestors"`
	AssetType               string          `json:"assetType"`
	AssetTypeLegacy         string          `json:"asset_type"`
	IamPolicy               json.RawMessage `json:"iamPolicy"`
	IamPolicyLegacy         json.RawMessage `json:"iam_policy"`
	Resource                json.RawMessage `json:"resource"`
}

// Assets array of Asset
type Assets []Asset

// Violations array of Violation
type Violations []Violation

// Violation from the "audit" rego policy in "audit.rego" module
type Violation struct {
	NonCompliance    NonCompliance     `json:"nonCompliance"`
	FunctionConfig   FunctionConfig    `json:"functionConfig"`
	ConstraintConfig ConstraintConfig  `json:"constraintConfig"`
	FeedMessage      FeedMessage       `json:"feedMessage"`
	RegoModules      map[string]string `json:"regoModules"`
}

// NonCompliance form the "deny" rego policy in a <templateName>.rego module
type NonCompliance struct {
	Message  string                 `json:"message"`
	Metadata map[string]interface{} `json:"metadata"`
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

// ConstraintMetadata Constraint's metadata
type ConstraintMetadata struct {
	Name        string                 `json:"name"`
	Annotations map[string]interface{} `json:"annotation"`
}

// Spec Constraint's specifications
type Spec struct {
	Severity   string                 `json:"severity"`
	Match      map[string]interface{} `json:"match"`
	Parameters map[string]interface{} `json:"parameters"`
}

// Parameters Constraint's settings
type Parameters map[string]json.RawMessage

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

// CompliantLog log entry when compliant
type CompliantLog struct {
	ComplianceStatus   ComplianceStatus `json:"complianceStatus"`
	AssetsJSONDocument json.RawMessage  `json:"assetsJSONDocument"`
}

// Initialize is to be executed in the init() function of the cloud function to optimize the cold start
func Initialize(ctx context.Context, global *Global) {
	global.ctx = ctx
	global.initFailed = false

	// err is pre-declared to avoid shadowing client.
	var err error
	var ok bool

	global.assetsCollectionID = os.Getenv("ASSETSCOLLECTIONID")
	global.environment = os.Getenv("ENVIRONMENT")
	global.functionName = os.Getenv("FUNCTION_NAME")
	global.ownerLabelKeyName = os.Getenv("OWNERLABELKEYNAME")
	global.projectID = os.Getenv("GCP_PROJECT")
	global.ramComplianceStatusTopicName = os.Getenv("STATUS_TOPIC")
	global.ramViolationTopicName = os.Getenv("VIOLATION_TOPIC")
	global.violationResolverLabelKeyName = os.Getenv("VIOLATIONRESOLVERLABELKEYNAME")

	log.Println("Function COLD START")
	if global.retryTimeOutSeconds, ok = ram.GetEnvVarInt64("RETRYTIMEOUTSECONDS"); !ok {
		return
	}
	if global.deploymentTime, ok = ram.GetEnvVarTime("DEPLOYMENT_TIME"); !ok {
		return
	}
	settingsFileContent, err := ioutil.ReadFile(settingsFileName)
	if err != nil {
		log.Printf("ERROR - ioutil.ReadFile: %v", err)
		global.initFailed = true
		return
	}
	err = json.Unmarshal(settingsFileContent, &global.settings)
	if err != nil {
		log.Printf("ERROR - json.Unmarshal: %v", err)
		global.initFailed = true
		return
	}

	global.assetsFolderPath = global.settings.WritabelOPAFolderPath + "/" + global.settings.AssetsFolderName
	global.assetsFilePath = global.assetsFolderPath + "/" + global.settings.AssetsFileName
	global.regoModulesFolderPath = global.settings.OPAFolderPath + "/" + global.settings.RegoModulesFolderName

	// services are initialized with context.Background() because it should
	// persist between function invocations.
	global.cloudresourcemanagerService, err = cloudresourcemanager.NewService(ctx)
	if err != nil {
		log.Printf("ERROR - cloudresourcemanager.NewService: %v", err)
		global.initFailed = true
		return
	}
	global.cloudresourcemanagerServiceV2, err = cloudresourcemanagerv2.NewService(ctx)
	if err != nil {
		log.Printf("ERROR - cloudresourcemanagerv2.NewService: %v", err)
		global.initFailed = true
		return
	}
	global.pubSubClient, err = pubsub.NewClient(ctx, global.projectID)
	if err != nil {
		log.Printf("ERROR - pubsub.NewClient: %v", err)
		global.initFailed = true
		return
	}
	global.firestoreClient, err = firestore.NewClient(global.ctx, global.projectID)
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

	var complianceStatus ComplianceStatus
	var compliantLog CompliantLog

	assetsJSONDocument, feedMessage, err := buildAssetsDocument(PubSubMessage, global)
	if err != nil {
		log.Printf("ERROR - buildAssetsDocument %v", err)
		return nil // NO RETRY
	}
	compliantLog.AssetsJSONDocument = assetsJSONDocument

	complianceStatus.AssetName = feedMessage.Asset.Name
	complianceStatus.AssetInventoryTimeStamp = feedMessage.Window.StartTime
	complianceStatus.AssetInventoryOrigin = feedMessage.Origin
	complianceStatus.RuleName = global.functionName
	complianceStatus.RuleDeploymentTimeStamp = global.deploymentTime
	if feedMessage.Deleted == true {
		complianceStatus.Deleted = feedMessage.Deleted
		// bool cannot be nil and have a zero value to false
		complianceStatus.Compliant = true
	} else {
		complianceStatus.Deleted = false
		resultSet, feedMessage, err := evalutateConstraints(assetsJSONDocument, feedMessage, global)
		if err != nil {
			log.Printf("ERROR - evalutateConstraints %v", err)
			return nil // NO RETRY
		}
		violations, err := inspectResultSet(resultSet, feedMessage, global)
		if err != nil {
			log.Printf("ERROR - inspectResultSet %v", err)
			return nil // NO RETRY
		}
		if len(violations) == 0 {
			complianceStatus.Compliant = true
		} else {
			complianceStatus.Compliant = false
			for i, violation := range violations {
				violationJSON, err := json.Marshal(violation)
				if err != nil {
					log.Printf("ERROR - json.Marshal(violation) %v", err)
					return nil // NO RETRY
				}
				log.Printf("NOT_COMPLIANT %s %s %v violationNum %d %s", complianceStatus.AssetName, complianceStatus.AssetInventoryOrigin, complianceStatus.AssetInventoryTimeStamp, i, string(violationJSON))
				err = publishPubSubMessage(violationJSON, global.ramViolationTopicName, global)
				if err != nil {
					return err // RETRY
				}
			}
		}
	}
	complianceStatusJSON, err := json.Marshal(complianceStatus)
	if err != nil {
		log.Printf("ERROR - json.Marshal(complianceStatus) %v", err)
		return nil // NO RETRY
	}
	err = publishPubSubMessage(complianceStatusJSON, global.ramComplianceStatusTopicName, global)
	if err != nil {
		return err // RETRY
	}
	compliantLog.ComplianceStatus = complianceStatus

	if complianceStatus.Compliant == true {
		CompliantLogJSON, err := json.Marshal(compliantLog)
		if err != nil {
			log.Printf("ERROR - json.Marshal(compliantLog) %v", err)
			return nil // NO RETRY
		}
		if complianceStatus.Deleted == true {
			log.Printf("DELETED %s %s %v %s", complianceStatus.AssetName, complianceStatus.AssetInventoryOrigin, complianceStatus.AssetInventoryTimeStamp, string(CompliantLogJSON))
		} else {
			log.Printf("COMPLIANT %s %s %v %s", complianceStatus.AssetName, complianceStatus.AssetInventoryOrigin, complianceStatus.AssetInventoryTimeStamp, string(CompliantLogJSON))
		}
	}
	return nil
}

func publishPubSubMessage(docJSON []byte, topicName string, global *Global) error {
	publishRequest := ram.PublishRequest{Topic: topicName}
	pubSubMessage := &pubsub.Message{
		Data: docJSON,
	}
	id, err := global.pubSubClient.Topic(publishRequest.Topic).Publish(global.ctx, pubSubMessage).Get(global.ctx)
	if err != nil {
		return fmt.Errorf("topic(%s).Publish.Get: %v", publishRequest.Topic, err)
	}
	// log.Printf("Published to topic %s, msg id: %v", topicName, id)
	_ = id
	return nil
}

// inspectResultSet explore rego query output and craft violation document
func inspectResultSet(resultSet rego.ResultSet, feedMessage FeedMessage, global *Global) (Violations, error) {
	var violations Violations
	var violation Violation

	regoModules := importRegoModulesCode(global)
	if regoModules == nil {
		return nil, fmt.Errorf("importRegoModulesCode was nil and should not")
	}

	Expressions := resultSet[0].Expressions
	if len(Expressions) != 0 {
		expressionValue := *Expressions[0]

		// log.Println("Rego Query: ", expressionValue.Text)
		// location := *expressionValue.Location
		// log.Printf("Position in query: Row %d Col %d\n", location.Row, location.Col)

		var valuesInterface interface{} = expressionValue.Value
		if values, ok := valuesInterface.([]interface{}); ok {
			for i := 0; i < len(values); i++ {
				var valueInterface interface{} = values[i]
				if value, ok := valueInterface.(map[string]interface{}); ok {
					violation.FunctionConfig.FunctionName = global.functionName
					violation.FunctionConfig.ProjectID = global.projectID
					violation.FunctionConfig.Environment = global.environment
					violation.FunctionConfig.DeploymentTime = global.deploymentTime
					violation.FeedMessage = feedMessage

					var violationInterface interface{} = value["violation"]
					if ruleViolation, ok := violationInterface.(map[string]interface{}); ok {
						var msgInterface interface{} = ruleViolation["msg"]
						if msg, ok := msgInterface.(string); ok {
							violation.NonCompliance.Message = msg
						}
						var detailsInterface interface{} = ruleViolation["details"]
						if details, ok := detailsInterface.(map[string]interface{}); ok {
							violation.NonCompliance.Metadata = details
						}
					}

					var constraintConfigInterface interface{} = value["constraint_config"]
					if constraintConfig, ok := constraintConfigInterface.(map[string]interface{}); ok {
						var apiVersionInterface interface{} = constraintConfig["apiVersion"]
						if apiVersion, ok := apiVersionInterface.(string); ok {
							violation.ConstraintConfig.APIVersion = apiVersion
						}
						var kindInterface interface{} = constraintConfig["kind"]
						if kind, ok := kindInterface.(string); ok {
							violation.ConstraintConfig.Kind = kind
						}
						var metadataInterface interface{} = constraintConfig["metadata"]
						if metadata, ok := metadataInterface.(map[string]interface{}); ok {
							var nameInterface interface{} = metadata["name"]
							if name, ok := nameInterface.(string); ok {
								violation.ConstraintConfig.Metadata.Name = name
							}
							var annotationsInterface interface{} = metadata["annotations"]
							if annotations, ok := annotationsInterface.(map[string]interface{}); ok {
								violation.ConstraintConfig.Metadata.Annotations = annotations
							}
						}

						var specInterface interface{} = constraintConfig["spec"]
						if spec, ok := specInterface.(map[string]interface{}); ok {
							var severityInterface interface{} = spec["severity"]
							if severity, ok := severityInterface.(string); ok {
								violation.ConstraintConfig.Spec.Severity = severity
							}
							var matchInterface interface{} = spec["match"]
							if match, ok := matchInterface.(map[string]interface{}); ok {
								violation.ConstraintConfig.Spec.Match = match
							}
							var parametersInterface interface{} = spec["parameters"]
							if parameters, ok := parametersInterface.(map[string]interface{}); ok {
								violation.ConstraintConfig.Spec.Parameters = parameters
							}
						}
					}
					violation.RegoModules = regoModules
				}
				violations = append(violations, violation)
			}
		}
	}
	return violations, nil
}

// importRegoModulesCode read regoModule code to be added in violation for logging / troubleshooting purposes
func importRegoModulesCode(global *Global) map[string]string {
	regoModules := make(map[string]string)
	files, err := ioutil.ReadDir(global.regoModulesFolderPath)
	if err != nil {
		log.Printf("ERROR - ioutil.ReadDir %v", err)
		return nil
	}

	for _, file := range files {
		regoCode, err := ioutil.ReadFile(global.regoModulesFolderPath + "/" + file.Name())
		if err != nil {
			log.Printf("ERROR - ioutil.ReadFile %v", err)
			return nil
		}
		regoModules[file.Name()] = string(regoCode)
	}
	return regoModules
}

// evalutateConstraints audit assets data to rego rules
func evalutateConstraints(assetsJSONDocument []byte, feedMessage FeedMessage, global *Global) (rego.ResultSet, FeedMessage, error) {
	var resultSet rego.ResultSet
	if _, err := os.Stat(global.assetsFilePath); os.IsExist(err) {
		// log.Println("Found ", assetsFilePath)
		err := os.Remove(global.assetsFilePath)
		if err != nil {
			return resultSet, feedMessage, fmt.Errorf("os.Remove(assetsFilePath) %v", err)
		}
	}
	if _, err := os.Stat(global.assetsFolderPath); os.IsNotExist(err) {
		err = os.MkdirAll(global.assetsFolderPath, 0755)
		if err != nil {
			return resultSet, feedMessage, fmt.Errorf("os.MkdirAll(assetsFolderPath, 0755) %v", err)
		}
	}
	err := ioutil.WriteFile(global.assetsFilePath, assetsJSONDocument, 0644)
	if err != nil {
		return resultSet, feedMessage, fmt.Errorf("ioutil.WriteFile(assetsFilePath, assetsJSONDocument, 0644) %v", err)
	}

	ctx := context.Background()
	rego := rego.New(rego.Query("audit"),
		rego.Load([]string{global.settings.OPAFolderPath, global.settings.WritabelOPAFolderPath}, nil),
		rego.Package("validator.gcp.lib"))

	resultSet, err = rego.Eval(ctx)
	if err != nil {
		return resultSet, feedMessage, fmt.Errorf("rego.Eval %v", err)
	}
	return resultSet, feedMessage, nil
}

// buildAssetsDocument
func buildAssetsDocument(pubSubMessage ram.PubSubMessage, global *Global) ([]byte, FeedMessage, error) {
	var feedMessage FeedMessage
	var assetsJSONDocument []byte
	var assets Assets

	err := json.Unmarshal(pubSubMessage.Data, &feedMessage)
	if err != nil {
		return assetsJSONDocument, feedMessage, fmt.Errorf("json.Unmarshal(pubSubMessage.Data, &feedMessage) %v", err)
	}

	if feedMessage.Origin == "" {
		feedMessage.Origin = "real-time"
	}

	feedMessage.Asset.AncestryPath = ram.BuildAncestryPath(feedMessage.Asset.Ancestors)
	feedMessage.Asset.AncestorsDisplayName = ram.BuildAncestorsDisplayName(global.ctx, feedMessage.Asset.Ancestors, global.assetsCollectionID, global.firestoreClient, global.cloudresourcemanagerService, global.cloudresourcemanagerServiceV2)
	feedMessage.Asset.AncestryPathDisplayName = ram.BuildAncestryPath(feedMessage.Asset.AncestorsDisplayName)

	feedMessage.Asset.Owner, _ = ram.GetAssetContact(global.ownerLabelKeyName, feedMessage.Asset.Resource)
	feedMessage.Asset.ViolationResolver, _ = ram.GetAssetContact(global.violationResolverLabelKeyName, feedMessage.Asset.Resource)
	// Duplicate fileds into fieldLegacy for compatibility with existing policy library templates
	feedMessage.Asset.IamPolicyLegacy = feedMessage.Asset.IamPolicy
	feedMessage.Asset.AssetTypeLegacy = feedMessage.Asset.AssetType
	feedMessage.Asset.AncestryPathLegacy = feedMessage.Asset.AncestryPath

	assets = append(assets, feedMessage.Asset)
	assetsJSONDocument, err = json.Marshal(assets)
	if err != nil {
		return assetsJSONDocument, feedMessage, fmt.Errorf("json.Marshal(assets) %v", err)
	}

	// log.Println("assetsJSONDocument", string(assetsJSONDocument))
	return assetsJSONDocument, feedMessage, nil
}
