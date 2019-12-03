package mock

type StatusMetricsStub struct {
	StatusMetricsMapWithoutP2PCalled func() map[string]interface{}
	StatusP2pMetricsMapCalled        func() map[string]interface{}
	IsInterfaceNilCalled             func() bool
}

func (nds *StatusMetricsStub) StatusMetricsMapWithoutP2P() map[string]interface{} {
	return nds.StatusMetricsMapWithoutP2PCalled()
}

func (nds *StatusMetricsStub) StatusP2pMetricsMap() map[string]interface{} {
	return nds.StatusP2pMetricsMapCalled()
}

func (nds *StatusMetricsStub) IsInterfaceNil() bool {
	return nds == nil
}
