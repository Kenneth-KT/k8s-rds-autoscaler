package autoscaler

type ScalingCapacity struct {
	Scale           string `json:"scale,omitempty"`
	ConnectionLimit int    `json:"limit,omitempty"`
}
