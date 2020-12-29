package autoscaler

import (
	"strconv"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/rds"
)

type RdsScaleManager struct {
	client       *rds.RDS
	timeout      int
	dbIdentifier string
}

func scaleAsInt64(sc *ScalingCapacity) int64 {
	intVal, err := strconv.Atoi(sc.Scale)
	if err != nil {
		panic(err.Error())
	}

	return int64(intVal)
}

func NewRdsScaleManager(
	session *session.Session,
	timeout int,
	dbIdentifier string,
) *RdsScaleManager {
	return &RdsScaleManager{
		client:       rds.New(session),
		timeout:      timeout,
		dbIdentifier: dbIdentifier,
	}
}

func (mgr *RdsScaleManager) setScale(desiredScale *ScalingCapacity) error {
	timeoutAction := "ForceApplyCapacityChange"
	scaleInt64 := scaleAsInt64(desiredScale)
	secondsBeforeTimeout := int64(mgr.timeout)
	_, err := mgr.client.ModifyCurrentDBClusterCapacity(
		&rds.ModifyCurrentDBClusterCapacityInput{
			Capacity:             &scaleInt64,
			TimeoutAction:        &timeoutAction,
			SecondsBeforeTimeout: &secondsBeforeTimeout,
			DBClusterIdentifier:  &mgr.dbIdentifier,
		},
	)
	return err
}
