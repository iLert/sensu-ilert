package main

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/iLert/ilert-go"
	corev2 "github.com/sensu/sensu-go/api/core/v2"
	"github.com/sensu/sensu-plugin-sdk/sensu"
	"github.com/sensu/sensu-plugin-sdk/templates"
)

type HandlerConfig struct {
	sensu.PluginConfig
	authToken        string
	dedupKeyTemplate string
	statusMapJson    string
	summaryTemplate  string
	detailsTemplate  string
}

type eventStatusMap map[string][]uint32

var (
	config = HandlerConfig{
		PluginConfig: sensu.PluginConfig{
			Name:     "sensu-ilert-handler",
			Short:    "The Sensu Go Ilert handler for incident management",
			Keyspace: "sensu.io/plugins/sensu-ilert-handler/config",
		},
	}

	ilertConfigOptions = []*sensu.PluginConfigOption{
		{
			Path:      "token",
			Env:       "ILERT_SENSU_TOKEN",
			Argument:  "token",
			Shorthand: "t",
			Secret:    true,
			Usage:     "The Ilert API authentication token, can be set with ILERT_SENSU_TOKEN",
			Value:     &config.authToken,
			Default:   "",
		},
		{
			Path:      "dedup-key-template",
			Env:       "ILERT_DEDUP_KEY_TEMPLATE",
			Argument:  "dedup-key-template",
			Shorthand: "k",
			Usage:     "The Ilert deduplication key template, can be set with ILERT_DEDUP_KEY_TEMPLATE",
			Value:     &config.dedupKeyTemplate,
			Default:   "{{.Entity.Name}}-{{.Check.Name}}",
		},
		{
			Path:      "summary-template",
			Env:       "ILERT_SUMMARY_TEMPLATE",
			Argument:  "summary-template",
			Shorthand: "S",
			Usage:     "The template for the alert summary, can be set with ILERT_SUMMARY_TEMPLATE",
			Value:     &config.summaryTemplate,
			Default:   "{{.Entity.Name}}/{{.Check.Name}} : {{.Check.Output}}",
		},
		{
			Path:      "details-template",
			Env:       "ILERT_DETAILS_TEMPLATE",
			Argument:  "details-template",
			Shorthand: "d",
			Usage:     "The template for the alert details, can be set with ILERT_DETAILS_TEMPLATE (default full event JSON)",
			Value:     &config.detailsTemplate,
			Default:   "",
		},
	}
)

func main() {
	goHandler := sensu.NewGoHandler(&config.PluginConfig, ilertConfigOptions, checkArgs, manageIncident)
	goHandler.Execute()
}

func checkArgs(event *corev2.Event) error {

	if !event.HasCheck() {
		return fmt.Errorf("event does not contain check")
	}

	if len(config.authToken) == 0 {
		return fmt.Errorf("Authentication token is empty")
	}

	if len(config.authToken) == 0 {
		return fmt.Errorf("no auth token provided")
	}
	return nil
}

func manageIncident(event *corev2.Event) error {
	priority, err := getIlertPriority(event, config.statusMapJson)
	if err != nil {
		return err
	}
	log.Printf("Incident priority: %s", priority)

	summary, err := getSummary(event)
	if err != nil {
		return err
	}

	details, err := getDetails(event)
	if err != nil {
		return err
	}

	// "The maximum permitted length of PG event is 512 KB. Let's limit check output to 256KB to prevent triggering a failed send"
	if len(event.Check.Output) > 256000 {
		log.Printf("Warning Incident Payload Truncated!")
		event.Check.Output = "WARNING Truncated:i\n" + event.Check.Output[:256000] + "..."
	}

	eventType := ilert.EventTypes.Alert

	// 0 indicates OK
	// https://docs.sensu.io/sensu-go/latest/observability-pipeline/observe-schedule/checks/#check-specification
	if event.Check.Status == 0 {
		eventType = ilert.EventTypes.Resolve
	}

	dedupKey, err := getIlertDedupKey(event)
	if err != nil {
		return err
	}
	if len(dedupKey) == 0 {
		return fmt.Errorf("iLert dedup key is empty")
	}

	client := ilert.NewClient(ilert.WithRetry(10, 5*time.Second, 20*time.Second))

	ilertEvent := &ilert.Event{
		APIKey:      config.authToken,
		EventType:   eventType,
		Summary:     summary,
		IncidentKey: dedupKey,
		Details:     details,
	}

	input := &ilert.CreateEventInput{Event: ilertEvent}
	result, err := client.CreateEvent(input)
	if err != nil {
		if apiErr, ok := err.(*ilert.GenericAPIError); ok {
			if apiErr.Code == "NO_OPEN_INCIDENT_WITH_KEY" {
				return fmt.Errorf("WARN: %s", apiErr.Error())
			} else {
				return fmt.Errorf("ERROR: %s", apiErr.Error())
			}
		} else {
			return fmt.Errorf("ERROR: %s", err)
		}
	}

	log.Printf("Event (%s) submitted to Ilert, Code: %s, Incident Key: %s, URL: %s", eventType, result.EventResponse.ResponseCode, result.EventResponse.IncidentKey, result.EventResponse.IncidentURL)
	return nil
}

func getIlertDedupKey(event *corev2.Event) (string, error) {
	return templates.EvalTemplate("dedupKey", config.dedupKeyTemplate, event)
}

func getIlertPriority(event *corev2.Event, statusMapJson string) (string, error) {
	var statusMap map[uint32]string
	var err error

	if len(statusMapJson) > 0 {
		statusMap, err = parseStatusMap(statusMapJson)
		if err != nil {
			return "", err
		}
	}

	if len(statusMap) > 0 {
		status := event.Check.Status
		priority := statusMap[status]
		if len(priority) > 0 {
			return priority, nil
		}
	}

	// Default to these values is no status map is found
	// The default of ilert priority is High
	priority := "high"
	// if event.Check.Status < 3 {
	// 	priorities := []string{"low", "high", "critical"}
	// 	priority = priorities[event.Check.Status]
	// }

	return priority, nil
}

func parseStatusMap(statusMapJson string) (map[uint32]string, error) {
	validIlertSeverities := map[string]bool{"info": true, "critical": true, "warning": true, "error": true}

	statusMap := eventStatusMap{}
	err := json.Unmarshal([]byte(statusMapJson), &statusMap)
	if err != nil {
		return nil, err
	}

	// Reverse the map to key it on the status
	statusToSeverityMap := map[uint32]string{}
	for severity, statuses := range statusMap {
		if !validIlertSeverities[severity] {
			return nil, fmt.Errorf("invalid iLert severity: %s", severity)
		}
		for i := range statuses {
			statusToSeverityMap[uint32(statuses[i])] = severity
		}
	}

	return statusToSeverityMap, nil
}

func getSummary(event *corev2.Event) (string, error) {
	summary, err := templates.EvalTemplate("summary", config.summaryTemplate, event)
	if err != nil {
		return "", fmt.Errorf("failed to evaluate template %s: %v", config.summaryTemplate, err)
	}
	// "The maximum permitted length of this property is 1024 characters."
	if len(summary) > 1024 {
		summary = summary[:1024]
	}
	log.Printf("Incident Summary: %s", summary)
	return summary, nil
}

func getDetails(event *corev2.Event) (string, error) {
	var (
		details string
		err     error
	)

	if len(config.detailsTemplate) > 0 {
		details, err = templates.EvalTemplate("details", config.detailsTemplate, event)
		if err != nil {
			return "", fmt.Errorf("failed to evaluate template %s: %v", config.detailsTemplate, err)
		}
	} else {
		details = fmt.Sprintf("Incident from Sensu, from entity: %s, name: %s", event.Entity.Name, event.Check.Name)
	}

	log.Printf("Incident Details: %s", details)
	return details, nil
}
