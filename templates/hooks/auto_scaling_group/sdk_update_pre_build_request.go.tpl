if delta.DifferentAt("Spec.Tags") {
    name := string(*latest.ko.Spec.Name)
    err = syncTags(
        ctx, 
        desired.ko.Spec.Tags, latest.ko.Spec.Tags, 
        &name, convertToOrderedACKTags, rm.sdkapi, rm.metrics,
    )
    if err != nil {
        return nil, err
    }
}
if !delta.DifferentExcept("Spec.Tags", enabledMetricsDeltaPath) {
    ko := desired.ko.DeepCopy()
    latest.ko.Status.DeepCopyInto(&ko.Status)
    return &resource{ko}, rm.updateMetricsCollection(ctx, ko, latest, delta)
}
