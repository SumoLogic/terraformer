// Copyright 2026 The Terraformer Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package sumologic

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/GoogleCloudPlatform/terraformer/terraformutils"
	"github.com/iancoleman/strcase"
)

// MetricRuleGenerator generates Terraform resources for SumoLogic metric rules.
type MetricRuleGenerator struct {
	SumoLogicService
}

type metricRuleVariable struct {
	Name        string `json:"name"`
	TagSequence string `json:"tagSequence"`
}

type metricRule struct {
	Name               string               `json:"name"`
	MatchExpression    string               `json:"matchExpression"`
	VariablesToExtract []metricRuleVariable `json:"variablesToExtract"`
}

func (g *MetricRuleGenerator) fetchMetricRule(name string) (*metricRule, error) {
	baseURL := strings.TrimSuffix(g.GetArgs()["baseUrl"].(string), "/")
	accessID := g.GetArgs()["accessId"].(string)
	accessKey := g.GetArgs()["accessKey"].(string)

	url := fmt.Sprintf("%s/v1/metricsRules/%s", baseURL, name)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(accessID, accessKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close() //nolint:errcheck

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch metric rule '%s': status %d, body: %s", name, resp.StatusCode, string(body))
	}

	var rule metricRule
	if err := json.Unmarshal(body, &rule); err != nil {
		return nil, err
	}
	return &rule, nil
}

func (g *MetricRuleGenerator) getFilteredNames() []string {
	var names []string
	for _, f := range g.Filter {
		if f.FieldPath == "id" {
			names = append(names, f.AcceptableValues...)
		}
	}
	return names
}

func (g *MetricRuleGenerator) createResource(rule *metricRule) terraformutils.Resource {
	name := strcase.ToSnake(replaceSpaceAndDash(rule.Name))
	return terraformutils.NewSimpleResource(
		rule.Name,
		fmt.Sprintf("%s-%s", name, rule.Name),
		"sumologic_metric_rule",
		g.ProviderName,
		[]string{})
}

// InitResources fetches metric rules by name and registers them as Terraform resources.
func (g *MetricRuleGenerator) InitResources() error {
	names := g.getFilteredNames()
	if len(names) == 0 {
		return fmt.Errorf("metric_rule requires a --filter flag with at least one rule name, e.g. --filter=metric_rule=RuleName1:RuleName2")
	}

	for _, name := range names {
		rule, err := g.fetchMetricRule(name)
		if err != nil {
			return err
		}
		g.Resources = append(g.Resources, g.createResource(rule))
	}
	return nil
}
