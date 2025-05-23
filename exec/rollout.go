package exec

import (
	"github.com/netcracker/qubership-core-lib-go-paas-mediation-client/v8/entity"
)

type RolloutExecutor Pool[*entity.DeploymentRollout]

func NewFixedRolloutExecutor(parallelism int, bufferSize int) RolloutExecutor {
	return RolloutExecutor(NewFixedPool[*entity.DeploymentRollout](parallelism, bufferSize))
}
