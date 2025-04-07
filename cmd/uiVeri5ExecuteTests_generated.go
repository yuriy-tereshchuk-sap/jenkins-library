// Code generated by piper's step-generator. DO NOT EDIT.

package cmd

import (
	"fmt"
	"os"
	"reflect"
	"strings"
	"time"

	"github.com/SAP/jenkins-library/pkg/config"
	"github.com/SAP/jenkins-library/pkg/gcp"
	"github.com/SAP/jenkins-library/pkg/gcs"
	"github.com/SAP/jenkins-library/pkg/log"
	"github.com/SAP/jenkins-library/pkg/splunk"
	"github.com/SAP/jenkins-library/pkg/telemetry"
	"github.com/SAP/jenkins-library/pkg/validation"
	"github.com/bmatcuk/doublestar"
	"github.com/spf13/cobra"
)

type uiVeri5ExecuteTestsOptions struct {
	InstallCommand string   `json:"installCommand,omitempty"`
	RunCommand     string   `json:"runCommand,omitempty"`
	RunOptions     []string `json:"runOptions,omitempty"`
	TestOptions    string   `json:"testOptions,omitempty"`
	TestServerURL  string   `json:"testServerUrl,omitempty"`
}

type uiVeri5ExecuteTestsReports struct {
}

func (p *uiVeri5ExecuteTestsReports) persist(stepConfig uiVeri5ExecuteTestsOptions, gcpJsonKeyFilePath string, gcsBucketId string, gcsFolderPath string, gcsSubFolder string) {
	if gcsBucketId == "" {
		log.Entry().Info("persisting reports to GCS is disabled, because gcsBucketId is empty")
		return
	}
	log.Entry().Info("Uploading reports to Google Cloud Storage...")
	content := []gcs.ReportOutputParam{
		{FilePattern: "**/TEST-*.xml", ParamRef: "", StepResultType: "acceptance-test"},
		{FilePattern: "**/requirement.mapping", ParamRef: "", StepResultType: "requirement-mapping"},
		{FilePattern: "**/delivery.mapping", ParamRef: "", StepResultType: "delivery-mapping"},
	}

	gcsClient, err := gcs.NewClient(gcpJsonKeyFilePath, "")
	if err != nil {
		log.Entry().Errorf("creation of GCS client failed: %v", err)
		return
	}
	defer gcsClient.Close()
	structVal := reflect.ValueOf(&stepConfig).Elem()
	inputParameters := map[string]string{}
	for i := 0; i < structVal.NumField(); i++ {
		field := structVal.Type().Field(i)
		if field.Type.String() == "string" {
			paramName := strings.Split(field.Tag.Get("json"), ",")
			paramValue, _ := structVal.Field(i).Interface().(string)
			inputParameters[paramName[0]] = paramValue
		}
	}
	if err := gcs.PersistReportsToGCS(gcsClient, content, inputParameters, gcsFolderPath, gcsBucketId, gcsSubFolder, doublestar.Glob, os.Stat); err != nil {
		log.Entry().Errorf("failed to persist reports: %v", err)
	}
}

// UiVeri5ExecuteTestsCommand Executes UI5 e2e tests using uiVeri5
func UiVeri5ExecuteTestsCommand() *cobra.Command {
	const STEP_NAME = "uiVeri5ExecuteTests"

	metadata := uiVeri5ExecuteTestsMetadata()
	var stepConfig uiVeri5ExecuteTestsOptions
	var startTime time.Time
	var reports uiVeri5ExecuteTestsReports
	var logCollector *log.CollectorHook
	var splunkClient *splunk.Splunk
	telemetryClient := &telemetry.Telemetry{}

	var createUiVeri5ExecuteTestsCmd = &cobra.Command{
		Use:   STEP_NAME,
		Short: "Executes UI5 e2e tests using uiVeri5",
		Long:  `In this step the ([UIVeri5 tests](https://github.com/SAP/ui5-uiveri5)) are executed.`,
		PreRunE: func(cmd *cobra.Command, _ []string) error {
			startTime = time.Now()
			log.SetStepName(STEP_NAME)
			log.SetVerbose(GeneralConfig.Verbose)

			GeneralConfig.GitHubAccessTokens = ResolveAccessTokens(GeneralConfig.GitHubTokens)

			path, err := os.Getwd()
			if err != nil {
				return err
			}
			fatalHook := &log.FatalHook{CorrelationID: GeneralConfig.CorrelationID, Path: path}
			log.RegisterHook(fatalHook)

			err = PrepareConfig(cmd, &metadata, STEP_NAME, &stepConfig, config.OpenPiperFile)
			if err != nil {
				log.SetErrorCategory(log.ErrorConfiguration)
				return err
			}

			if len(GeneralConfig.HookConfig.SentryConfig.Dsn) > 0 {
				sentryHook := log.NewSentryHook(GeneralConfig.HookConfig.SentryConfig.Dsn, GeneralConfig.CorrelationID)
				log.RegisterHook(&sentryHook)
			}

			if len(GeneralConfig.HookConfig.SplunkConfig.Dsn) > 0 || len(GeneralConfig.HookConfig.SplunkConfig.ProdCriblEndpoint) > 0 {
				splunkClient = &splunk.Splunk{}
				logCollector = &log.CollectorHook{CorrelationID: GeneralConfig.CorrelationID}
				log.RegisterHook(logCollector)
			}

			if err = log.RegisterANSHookIfConfigured(GeneralConfig.CorrelationID); err != nil {
				log.Entry().WithError(err).Warn("failed to set up SAP Alert Notification Service log hook")
			}

			validation, err := validation.New(validation.WithJSONNamesForStructFields(), validation.WithPredefinedErrorMessages())
			if err != nil {
				return err
			}
			if err = validation.ValidateStruct(stepConfig); err != nil {
				log.SetErrorCategory(log.ErrorConfiguration)
				return err
			}

			return nil
		},
		Run: func(_ *cobra.Command, _ []string) {
			vaultClient := config.GlobalVaultClient()
			if vaultClient != nil {
				defer vaultClient.MustRevokeToken()
			}

			stepTelemetryData := telemetry.CustomData{}
			stepTelemetryData.ErrorCode = "1"
			handler := func() {
				reports.persist(stepConfig, GeneralConfig.GCPJsonKeyFilePath, GeneralConfig.GCSBucketId, GeneralConfig.GCSFolderPath, GeneralConfig.GCSSubFolder)
				config.RemoveVaultSecretFiles()
				stepTelemetryData.Duration = fmt.Sprintf("%v", time.Since(startTime).Milliseconds())
				stepTelemetryData.ErrorCategory = log.GetErrorCategory().String()
				stepTelemetryData.PiperCommitHash = GitCommit
				telemetryClient.SetData(&stepTelemetryData)
				telemetryClient.LogStepTelemetryData()
				if len(GeneralConfig.HookConfig.SplunkConfig.Dsn) > 0 {
					splunkClient.Initialize(GeneralConfig.CorrelationID,
						GeneralConfig.HookConfig.SplunkConfig.Dsn,
						GeneralConfig.HookConfig.SplunkConfig.Token,
						GeneralConfig.HookConfig.SplunkConfig.Index,
						GeneralConfig.HookConfig.SplunkConfig.SendLogs)
					splunkClient.Send(telemetryClient.GetData(), logCollector)
				}
				if len(GeneralConfig.HookConfig.SplunkConfig.ProdCriblEndpoint) > 0 {
					splunkClient.Initialize(GeneralConfig.CorrelationID,
						GeneralConfig.HookConfig.SplunkConfig.ProdCriblEndpoint,
						GeneralConfig.HookConfig.SplunkConfig.ProdCriblToken,
						GeneralConfig.HookConfig.SplunkConfig.ProdCriblIndex,
						GeneralConfig.HookConfig.SplunkConfig.SendLogs)
					splunkClient.Send(telemetryClient.GetData(), logCollector)
				}
				if GeneralConfig.HookConfig.GCPPubSubConfig.Enabled {
					err := gcp.NewGcpPubsubClient(
						vaultClient,
						GeneralConfig.HookConfig.GCPPubSubConfig.ProjectNumber,
						GeneralConfig.HookConfig.GCPPubSubConfig.IdentityPool,
						GeneralConfig.HookConfig.GCPPubSubConfig.IdentityProvider,
						GeneralConfig.CorrelationID,
						GeneralConfig.HookConfig.OIDCConfig.RoleID,
					).Publish(GeneralConfig.HookConfig.GCPPubSubConfig.Topic, telemetryClient.GetDataBytes())
					if err != nil {
						log.Entry().WithError(err).Warn("event publish failed")
					}
				}
			}
			log.DeferExitHandler(handler)
			defer handler()
			telemetryClient.Initialize(STEP_NAME)
			uiVeri5ExecuteTests(stepConfig, &stepTelemetryData)
			stepTelemetryData.ErrorCode = "0"
			log.Entry().Info("SUCCESS")
		},
	}

	addUiVeri5ExecuteTestsFlags(createUiVeri5ExecuteTestsCmd, &stepConfig)
	return createUiVeri5ExecuteTestsCmd
}

func addUiVeri5ExecuteTestsFlags(cmd *cobra.Command, stepConfig *uiVeri5ExecuteTestsOptions) {
	cmd.Flags().StringVar(&stepConfig.InstallCommand, "installCommand", `npm install @ui5/uiveri5 --global --quiet`, "The command that is executed to install the uiveri5 test tool.")
	cmd.Flags().StringVar(&stepConfig.RunCommand, "runCommand", `/home/node/.npm-global/bin/uiveri5`, "The command that is executed to start the tests.")
	cmd.Flags().StringSliceVar(&stepConfig.RunOptions, "runOptions", []string{`--seleniumAddress=http://localhost:4444/wd/hub`}, "Options to append to the runCommand, last parameter has to be path to conf.js (default if missing: ./conf.js).")
	cmd.Flags().StringVar(&stepConfig.TestOptions, "testOptions", os.Getenv("PIPER_testOptions"), "Deprecated and will result in an error if set. Please use runOptions instead. Split the testOptions string at the whitespaces when migrating it into a list of runOptions.")
	cmd.Flags().StringVar(&stepConfig.TestServerURL, "testServerUrl", os.Getenv("PIPER_testServerUrl"), "URL pointing to the deployment.")

	cmd.MarkFlagRequired("installCommand")
	cmd.MarkFlagRequired("runCommand")
	cmd.MarkFlagRequired("runOptions")
}

// retrieve step metadata
func uiVeri5ExecuteTestsMetadata() config.StepData {
	var theMetaData = config.StepData{
		Metadata: config.StepMetadata{
			Name:        "uiVeri5ExecuteTests",
			Aliases:     []config.Alias{},
			Description: "Executes UI5 e2e tests using uiVeri5",
		},
		Spec: config.StepSpec{
			Inputs: config.StepInputs{
				Resources: []config.StepResources{
					{Name: "buildDescriptor", Type: "stash"},
					{Name: "tests", Type: "stash"},
				},
				Parameters: []config.StepParameters{
					{
						Name:        "installCommand",
						ResourceRef: []config.ResourceReference{},
						Scope:       []string{"GENERAL", "PARAMETERS", "STAGES", "STEPS"},
						Type:        "string",
						Mandatory:   true,
						Aliases:     []config.Alias{},
						Default:     `npm install @ui5/uiveri5 --global --quiet`,
					},
					{
						Name:        "runCommand",
						ResourceRef: []config.ResourceReference{},
						Scope:       []string{"GENERAL", "PARAMETERS", "STAGES", "STEPS"},
						Type:        "string",
						Mandatory:   true,
						Aliases:     []config.Alias{},
						Default:     `/home/node/.npm-global/bin/uiveri5`,
					},
					{
						Name:        "runOptions",
						ResourceRef: []config.ResourceReference{},
						Scope:       []string{"PARAMETERS", "STAGES", "STEPS"},
						Type:        "[]string",
						Mandatory:   true,
						Aliases:     []config.Alias{},
						Default:     []string{`--seleniumAddress=http://localhost:4444/wd/hub`},
					},
					{
						Name:        "testOptions",
						ResourceRef: []config.ResourceReference{},
						Scope:       []string{"PARAMETERS", "STAGES", "STEPS"},
						Type:        "string",
						Mandatory:   false,
						Aliases:     []config.Alias{},
						Default:     os.Getenv("PIPER_testOptions"),
					},
					{
						Name:        "testServerUrl",
						ResourceRef: []config.ResourceReference{},
						Scope:       []string{"PARAMETERS", "STAGES", "STEPS"},
						Type:        "string",
						Mandatory:   false,
						Aliases:     []config.Alias{},
						Default:     os.Getenv("PIPER_testServerUrl"),
					},
				},
			},
			Containers: []config.Container{
				{Name: "uiVeri5", Image: "node:lts-bookworm", EnvVars: []config.EnvVar{{Name: "no_proxy", Value: "localhost,selenium,$no_proxy"}, {Name: "NO_PROXY", Value: "localhost,selenium,$NO_PROXY"}}, WorkingDir: "/home/node"},
			},
			Sidecars: []config.Container{
				{Name: "selenium", Image: "selenium/standalone-chrome", EnvVars: []config.EnvVar{{Name: "NO_PROXY", Value: "localhost,selenium,$NO_PROXY"}, {Name: "no_proxy", Value: "localhost,selenium,$no_proxy"}}},
			},
			Outputs: config.StepOutputs{
				Resources: []config.StepResources{
					{
						Name: "reports",
						Type: "reports",
						Parameters: []map[string]interface{}{
							{"filePattern": "**/TEST-*.xml", "type": "acceptance-test"},
							{"filePattern": "**/requirement.mapping", "type": "requirement-mapping"},
							{"filePattern": "**/delivery.mapping", "type": "delivery-mapping"},
						},
					},
				},
			},
		},
	}
	return theMetaData
}
