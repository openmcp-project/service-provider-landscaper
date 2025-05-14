package v1alpha1

const (
	LandscaperDomain           = "landscaper.services.openmcp.cloud"
	LandscaperFinalizer        = LandscaperDomain + "/finalizer"
	LandscaperOperation        = LandscaperDomain + "/operation"
	OperationReconcile         = "reconcile"
	ProviderConfigTypeLabel    = LandscaperDomain + "/providertype"
	DefaultProviderConfigValue = "default"
)
