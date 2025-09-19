package v1alpha2

const (
	LandscaperDomain           = "landscaper.services.openmcp.cloud"
	LandscaperProviderName     = "provider." + LandscaperDomain
	LandscaperFinalizer        = LandscaperDomain + "/finalizer"
	LandscaperOperation        = LandscaperDomain + "/operation"
	OperationReconcile         = "reconcile"
	ProviderConfigTypeLabel    = LandscaperDomain + "/providertype"
	DefaultProviderConfigValue = "default"
)
