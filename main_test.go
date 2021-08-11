package main

import (
	"encoding/json"
	"testing"

	corev2 "github.com/sensu/sensu-go/api/core/v2"
	"github.com/stretchr/testify/assert"
)

var (
	eventWithStatus = corev2.Event{
		Check: &corev2.Check{
			Status: 10,
		},
	}
)

func Test_ParseStatusMap_Success(t *testing.T) {
	json := "{\"info\":[130,10],\"error\":[4]}"

	statusMap, err := parseStatusMap(json)
	assert.Nil(t, err)
	assert.Equal(t, 3, len(statusMap))
	assert.Equal(t, "info", statusMap[130])
	assert.Equal(t, "info", statusMap[10])
	assert.Equal(t, "error", statusMap[4])
}

func Test_ParseStatusMap_EmptyStatus(t *testing.T) {
	json := "{\"info\":[130,10],\"error\":[]}"

	statusMap, err := parseStatusMap(json)
	assert.Nil(t, err)
	assert.Equal(t, 2, len(statusMap))
	assert.Equal(t, "info", statusMap[130])
	assert.Equal(t, "info", statusMap[10])
	assert.Equal(t, "", statusMap[4])
}

func Test_ParseStatusMap_InvalidJson(t *testing.T) {
	json := "{\"info\":[130,10],\"error:[]}"

	statusMap, err := parseStatusMap(json)
	assert.NotNil(t, err)
	assert.EqualError(t, err, "unexpected end of JSON input")
	assert.Nil(t, statusMap)
}

func Test_GetIlertPriority_Success(t *testing.T) {
	statusMapJson := "{\"info\":[130,10],\"error\":[4]}"

	eventWithStatus.Check.Status = 10
	ilertPriority, err := getIlertPriority(&eventWithStatus, statusMapJson)
	assert.Nil(t, err)
	assert.Equal(t, "info", ilertPriority)
}

func Test_GetIlertDedupKey(t *testing.T) {
	event := corev2.FixtureEvent("foo", "bar")
	config.dedupKeyTemplate = "{{.Entity.Name}}-{{.Check.Name}}"

	dedupKey, err := getIlertDedupKey(event)
	assert.Nil(t, err)
	assert.Equal(t, "foo-bar", dedupKey)
}

func Test_GetSummary(t *testing.T) {
	event := corev2.FixtureEvent("foo", "bar")
	config.summaryTemplate = "{{.Entity.Name}}-{{.Check.Name}}"

	summary, err := getSummary(event)
	assert.Nil(t, err)
	assert.Equal(t, "foo-bar", summary)
}

func Test_GetDetailsJSON(t *testing.T) {
	event := corev2.FixtureEvent("foo", "bar")
	config.detailsTemplate = ""

	details, err := getDetails(event)
	assert.Nil(t, err)
	assert.Equal(t, "Incident from Sensu, from entity: foo, name: bar", details)
	_, err = json.Marshal(details)
	assert.Nil(t, err)
}

func Test_GetDetailsTemplate(t *testing.T) {
	event := corev2.FixtureEvent("foo", "bar")
	config.detailsTemplate = "{{.Entity.Name}}-{{.Check.Name}}"

	details, err := getDetails(event)
	assert.Nil(t, err)
	assert.Equal(t, "foo-bar", details)
}
