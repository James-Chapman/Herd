package commander

type Status int

const (
	WORKER_ONLINE  Status = iota // 0
	WORKER_OFFLINE               // 1
)

type WorkerData struct {
	fqdn        string
	networkErrs int
	status      Status
}

// WorkerMap is a map of worker nodes and info related to each node
type WorkerMap map[string]*WorkerData

func (wm WorkerMap) AddWorker(server string) {
	_, found := wm[server]
	if !found {
		WorkersMtx.Lock()
		wm[server] = &WorkerData{server, 0, WORKER_ONLINE}
		WorkersMtx.Unlock()
	}
}

func (wm WorkerMap) AddNetError(server string) {
	_, found := wm[server]
	if found {
		WorkersMtx.Lock()
		pWorkerData := wm[server]
		pWorkerData.networkErrs += 1
		WorkersMtx.Unlock()
	}
}

func (wm WorkerMap) ResetNetError(server string) {
	_, found := wm[server]
	if found {
		WorkersMtx.Lock()
		pWorkerData := wm[server]
		pWorkerData.networkErrs = 0
		WorkersMtx.Unlock()
	}
}

func (wm WorkerMap) GetNetErrors(server string) int {
	_, found := wm[server]
	if found {
		pWorkerData := wm[server]
		return pWorkerData.networkErrs
	}
	return -1
}

func (wm WorkerMap) GetStatus(server string) Status {
	_, found := wm[server]
	if found {
		pWorkerData := wm[server]
		return pWorkerData.status
	}
	return WORKER_OFFLINE
}

func (wm WorkerMap) SetStatus(server string, stat Status) {
	_, found := wm[server]
	if found {
		WorkersMtx.Lock()
		pWorkerData := wm[server]
		pWorkerData.status = stat
		WorkersMtx.Unlock()
	}
}
