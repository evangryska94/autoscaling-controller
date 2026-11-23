// Copyright Amazon.com Inc. or its affiliates. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License"). You may
// not use this file except in compliance with the License. A copy of the
// License is located at
//
//     http://aws.amazon.com/apache2.0/
//
// or in the "license" file accompanying this file. This file is distributed
// on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either
// express or implied. See the License for the specific language governing
// permissions and limitations under the License.

package auto_scaling_group

import (
	"context"

	ackcompare "github.com/aws-controllers-k8s/runtime/pkg/compare"
	ackcondition "github.com/aws-controllers-k8s/runtime/pkg/condition"
	"github.com/aws/aws-sdk-go-v2/aws"
	svcsdk "github.com/aws/aws-sdk-go-v2/service/autoscaling"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/sets"

	svcapitypes "github.com/aws-controllers-k8s/autoscaling-controller/apis/v1alpha1"
	"github.com/aws-controllers-k8s/autoscaling-controller/pkg/tags"
)

var syncTags = tags.Tags

// Use the Kubernetes field name so IgnoreFieldDrift can match spec.enabledMetrics
// by case folding. The Go spec field is EnabledMetricNames.
const enabledMetricsDeltaPath = "Spec.EnabledMetrics"

func requestedMetricNames(metrics []*string) sets.Set[string] {
	names := sets.New[string]()
	for _, metric := range metrics {
		if metric != nil {
			names.Insert(*metric)
		}
	}
	return names
}

func enabledMetricNames(metrics []*svcapitypes.EnabledMetric) sets.Set[string] {
	names := sets.New[string]()
	for _, metric := range metrics {
		if metric != nil && metric.Metric != nil {
			names.Insert(*metric.Metric)
		}
	}
	return names
}

func (rm *resourceManager) updateMetricsCollection(
	ctx context.Context,
	ko *svcapitypes.AutoScalingGroup,
	latest *resource,
	delta *ackcompare.Delta,
) error {
	if !delta.DifferentAt(enabledMetricsDeltaPath) {
		return nil
	}
	desired := requestedMetricNames(ko.Spec.EnabledMetricNames)
	current := enabledMetricNames(latest.ko.Status.EnabledMetrics)
	groupName := latest.ko.Spec.Name

	// Neither API may receive an empty list because AWS would apply it to all
	// metrics. Enable additions first so an invalid metric leaves existing
	// collection in place.
	if additions := desired.Difference(current); len(additions) > 0 {
		_, err := rm.sdkapi.EnableMetricsCollection(ctx, &svcsdk.EnableMetricsCollectionInput{
			AutoScalingGroupName: groupName,
			Granularity:          aws.String("1Minute"),
			Metrics:              sets.List(additions),
		})
		rm.metrics.RecordAPICall("UPDATE", "EnableMetricsCollection", err)
		if err != nil {
			return err
		}
	}

	if removals := current.Difference(desired); len(removals) > 0 {
		_, err := rm.sdkapi.DisableMetricsCollection(ctx, &svcsdk.DisableMetricsCollectionInput{
			AutoScalingGroupName: groupName,
			Metrics:              sets.List(removals),
		})
		rm.metrics.RecordAPICall("UPDATE", "DisableMetricsCollection", err)
		if err != nil {
			return err
		}
	}
	// Let the next reconciliation read the metrics after AWS processes the change.
	ackcondition.SetSynced(&resource{ko}, corev1.ConditionFalse, nil, nil)
	return nil
}
