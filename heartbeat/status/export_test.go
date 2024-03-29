package status

// NewMetricsUpdaterWithoutGoRoutineStart -
func NewMetricsUpdaterWithoutGoRoutineStart(args ArgsMetricsUpdater) (*metricsUpdater, error) {
	err := checkArgs(args)
	if err != nil {
		return nil, err
	}

	updater := &metricsUpdater{
		peerAuthenticationCacher:            args.PeerAuthenticationCacher,
		heartbeatMonitor:                    args.HeartbeatMonitor,
		heartbeatSenderInfoProvider:         args.HeartbeatSenderInfoProvider,
		appStatusHandler:                    args.AppStatusHandler,
		timeBetweenConnectionsMetricsUpdate: args.TimeBetweenConnectionsMetricsUpdate,
		heartbeatV1DisableEpoch:             args.HeartbeatV1DisableEpoch,
	}

	args.EpochNotifier.RegisterNotifyHandler(updater)
	args.PeerAuthenticationCacher.RegisterHandler(updater.onAddedPeerAuthenticationMessage, "metricsUpdater")

	return updater, nil
}
