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

package monitor

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	pubsub "cloud.google.com/go/pubsub/apiv1"
	"github.com/BrunoReboul/ram/utilities/cai"
	"github.com/BrunoReboul/ram/utilities/ffo"
	"github.com/BrunoReboul/ram/utilities/gcf"
	"github.com/BrunoReboul/ram/utilities/gps"
	"github.com/BrunoReboul/ram/utilities/solution"
	"github.com/open-policy-agent/opa/rego"
	"google.golang.org/api/cloudresourcemanager/v1"
	cloudresourcemanagerv2 "google.golang.org/api/cloudresourcemanager/v2"
	pubsubpb "google.golang.org/genproto/googleapis/pubsub/v1"
)

// Global structure for global variables to optimize the cloud function performances
type Global struct {
	ctx                           context.Context
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
	opaFolderPath                 string
	ownerLabelKeyName             string
	projectID                     string
	pubsubPublisherClient         *pubsub.PublisherClient
	ramComplianceStatusTopicName  string
	ramViolationTopicName         string
	regoModulesFolderPath         string
	violationResolverLabelKeyName string
	writabelOPAFolderPath         string
}

// feedMessage Cloud Asset Inventory feed message
type feedMessage struct {
	Asset   asset      `json:"asset"`
	Window  cai.Window `json:"window"`
	Deleted bool       `json:"deleted"`
	Origin  string     `json:"origin"`
}

// asset Cloud Asset Metadata
// Duplicate "iamPolicy" and "assetType en ensure compatibility beetween format in CAI feed, aka real time, and CAI Export aka batch
type asset struct {
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

// assets slice of asset
type assets []asset

// violations array of violation
type violations []violation

// violation from the "audit" rego policy in "audit.rego" module
type violation struct {
	NonCompliance    nonCompliance     `json:"nonCompliance"`
	FunctionConfig   functionConfig    `json:"functionConfig"`
	ConstraintConfig constraintConfig  `json:"constraintConfig"`
	FeedMessage      feedMessage       `json:"feedMessage"`
	RegoModules      map[string]string `json:"regoModules"`
}

// nonCompliance form the "deny" rego policy in a <templateName>.rego module
type nonCompliance struct {
	Message  string                 `json:"message"`
	Metadata map[string]interface{} `json:"metadata"`
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

// constraintMetadata Constraint's metadata
type constraintMetadata struct {
	Name        string                 `json:"name"`
	Annotations map[string]interface{} `json:"annotation"`
}

// spec Constraint's specifications
type spec struct {
	Severity   string                 `json:"severity"`
	Match      map[string]interface{} `json:"match"`
	Parameters map[string]interface{} `json:"parameters"`
}

// compliantLog log entry when compliant
type compliantLog struct {
	ComplianceStatus   ComplianceStatus `json:"complianceStatus"`
	AssetsJSONDocument json.RawMessage  `json:"assetsJSONDocument"`
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

// Initialize is to be executed in the init() function of the cloud function to optimize the cold start
func Initialize(ctx context.Context, global *Global) (err error) {
	global.ctx = ctx

	var instanceDeployment InstanceDeployment

	log.Println("Function COLD START")
	err = ffo.ReadUnmarshalYAML(solution.PathToFunctionCode+solution.SettingsFileName, &instanceDeployment)
	if err != nil {
		return fmt.Errorf("ERROR - ReadUnmarshalYAML %s %v", solution.SettingsFileName, err)
	}

	assetsFileName := instanceDeployment.Settings.Service.AssetsFileName
	assetsFolderName := instanceDeployment.Settings.Service.AssetsFolderName
	global.assetsCollectionID = instanceDeployment.Core.SolutionSettings.Hosting.FireStore.CollectionIDs.Assets
	global.deploymentTime = instanceDeployment.Settings.Instance.DeploymentTime
	global.environment = instanceDeployment.Core.EnvironmentName
	global.functionName = instanceDeployment.Core.InstanceName
	global.opaFolderPath = instanceDeployment.Settings.Service.OPAFolderPath
	global.ownerLabelKeyName = instanceDeployment.Core.SolutionSettings.Monitoring.LabelKeyNames.Owner
	global.projectID = instanceDeployment.Core.SolutionSettings.Hosting.ProjectID
	global.ramComplianceStatusTopicName = instanceDeployment.Core.SolutionSettings.Hosting.Pubsub.TopicNames.RAMComplianceStatus
	global.ramViolationTopicName = instanceDeployment.Core.SolutionSettings.Hosting.Pubsub.TopicNames.RAMViolation
	global.retryTimeOutSeconds = instanceDeployment.Settings.Service.GCF.RetryTimeOutSeconds
	global.violationResolverLabelKeyName = instanceDeployment.Core.SolutionSettings.Monitoring.LabelKeyNames.ViolationResolver
	global.writabelOPAFolderPath = instanceDeployment.Settings.Service.WritabelOPAFolderPath
	regoModulesFolderName := instanceDeployment.Settings.Service.RegoModulesFolderName

	global.assetsFolderPath = global.writabelOPAFolderPath + "/" + assetsFolderName
	global.assetsFilePath = global.assetsFolderPath + "/" + assetsFileName
	global.regoModulesFolderPath = global.opaFolderPath + "/" + regoModulesFolderName

	// services are initialized with context.Background() because it should
	// persist between function invocations.
	global.cloudresourcemanagerService, err = cloudresourcemanager.NewService(ctx)
	if err != nil {
		return fmt.Errorf("ERROR - cloudresourcemanager.NewService: %v", err)
	}
	global.cloudresourcemanagerServiceV2, err = cloudresourcemanagerv2.NewService(ctx)
	if err != nil {
		return fmt.Errorf("ERROR - cloudresourcemanagerv2.NewService: %v", err)
	}
	global.pubsubPublisherClient, err = pubsub.NewPublisherClient(global.ctx)
	if err != nil {
		return fmt.Errorf("ERROR - global.pubsubPublisherClient: %v", err)
	}
	global.firestoreClient, err = firestore.NewClient(global.ctx, global.projectID)
	if err != nil {
		return fmt.Errorf("ERROR - firestore.NewClient: %v", err)
	}
	return nil
}

// EntryPoint is the function to be executed for each cloud function occurence
func EntryPoint(ctxEvent context.Context, PubSubMessage gps.PubSubMessage, global *Global) error {
	// log.Println(string(PubSubMessage.Data))
	if ok, _, err := gcf.IntialRetryCheck(ctxEvent, global.retryTimeOutSeconds); !ok {
		return err
	}
	// log.Printf("EventType %s EventID %s Resource %s Timestamp %v", metadata.EventType, metadata.EventID, metadata.Resource.Type, metadata.Timestamp)

	if strings.Contains(string(PubSubMessage.Data), "You have successfully configured real time feed") {
		log.Printf("Ignored pubsub message: %s", string(PubSubMessage.Data))
		return nil // NO RETRY
	}

	var complianceStatus ComplianceStatus
	var compliantLog compliantLog

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
	var pubSubMessage pubsubpb.PubsubMessage
	pubSubMessage.Data = docJSON

	var pubsubMessages []*pubsubpb.PubsubMessage
	pubsubMessages = append(pubsubMessages, &pubSubMessage)

	var publishRequest pubsubpb.PublishRequest
	publishRequest.Topic = fmt.Sprintf("projects/%s/topics/%s", global.projectID, topicName)
	publishRequest.Messages = pubsubMessages

	pubsubResponse, err := global.pubsubPublisherClient.Publish(global.ctx, &publishRequest)
	if err != nil {
		return fmt.Errorf("global.pubsubPublisherClient.Publish: %v", err) // RETRY
	}

	log.Printf("Published to topic %s, msg ids: %v", topicName, pubsubResponse.MessageIds)
	_ = pubsubResponse
	return nil
}

// inspectResultSet explore rego query output and craft violation document
func inspectResultSet(resultSet rego.ResultSet, feedMessage feedMessage, global *Global) (violations, error) {
	var violations violations
	var violation violation

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
func evalutateConstraints(assetsJSONDocument []byte, feedMessage feedMessage, global *Global) (rego.ResultSet, feedMessage, error) {
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
		rego.Load([]string{global.opaFolderPath, global.writabelOPAFolderPath}, nil),
		rego.Package("validator.gcp.lib"))

	resultSet, err = rego.Eval(ctx)
	if err != nil {
		return resultSet, feedMessage, fmt.Errorf("rego.Eval %v", err)
	}
	return resultSet, feedMessage, nil
}

// buildAssetsDocument
func buildAssetsDocument(pubSubMessage gps.PubSubMessage, global *Global) ([]byte, feedMessage, error) {
	var feedMessage feedMessage
	var assetsJSONDocument []byte
	var assets assets

	err := json.Unmarshal(pubSubMessage.Data, &feedMessage)
	if err != nil {
		log.Printf("ERROR - pubSubMessage.Data cannot be UnMarshalled as a feed %s", string(pubSubMessage.Data))
		return assetsJSONDocument, feedMessage, fmt.Errorf("json.Unmarshal(pubSubMessage.Data, &feedMessage) %v", err)
	}

	if feedMessage.Origin == "" {
		feedMessage.Origin = "real-time"
	}

	feedMessage.Asset.AncestryPath = cai.BuildAncestryPath(feedMessage.Asset.Ancestors)
	feedMessage.Asset.AncestorsDisplayName = cai.BuildAncestorsDisplayName(global.ctx, feedMessage.Asset.Ancestors, global.assetsCollectionID, global.firestoreClient, global.cloudresourcemanagerService, global.cloudresourcemanagerServiceV2)
	feedMessage.Asset.AncestryPathDisplayName = cai.BuildAncestryPath(feedMessage.Asset.AncestorsDisplayName)

	feedMessage.Asset.Owner, _ = cai.GetAssetLabelValue(global.ownerLabelKeyName, feedMessage.Asset.Resource)
	feedMessage.Asset.ViolationResolver, _ = cai.GetAssetLabelValue(global.violationResolverLabelKeyName, feedMessage.Asset.Resource)
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
