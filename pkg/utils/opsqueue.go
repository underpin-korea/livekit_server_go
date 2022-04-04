package utils

import (
	"sync"

	"github.com/underpin-korea/protocol/logger"
)

const (
	MaxOps = 50
)

type OpsQueue struct {
	logger logger.Logger

	lock      sync.RWMutex
	ops       chan func()
	isStopped bool
}

func NewOpsQueue(logger logger.Logger) *OpsQueue {
	return &OpsQueue{
		logger: logger,
		ops:    make(chan func(), MaxOps),
	}
}

func (oq *OpsQueue) SetLogger(logger logger.Logger) {
	oq.logger = logger
}

func (oq *OpsQueue) Start() {
	go oq.process()
}

func (oq *OpsQueue) Stop() {
	oq.lock.Lock()
	if oq.isStopped {
		oq.lock.Unlock()
		return
	}

	oq.isStopped = true
	close(oq.ops)
	oq.lock.Unlock()
}

func (oq *OpsQueue) Enqueue(op func()) {
	oq.lock.RLock()
	if oq.isStopped {
		oq.lock.RUnlock()
		return
	}

	select {
	case oq.ops <- op:
	default:
		oq.logger.Warnw("ops queue full", nil)
	}
	oq.lock.RUnlock()
}

func (oq *OpsQueue) process() {
	for op := range oq.ops {
		op()
	}
}
