if err = rm.updateMetricsCollection(ctx, ko, latest, delta); err != nil {
    return &resource{ko}, err
}
