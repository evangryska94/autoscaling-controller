if ko.Spec.Tags != nil || ko.Spec.EnabledMetricNames != nil {
    ackcondition.SetSynced(&resource{ko}, corev1.ConditionFalse, nil, nil)
}
