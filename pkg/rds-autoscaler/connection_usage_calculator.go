package autoscaler

import (
	"context"
	"fmt"
	"strconv"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const CONNECTION_USAGE_ANNOTATION = "rds-autoscaler/connections-usage"

type ConnectionUsageCalculatorPodsSelector struct {
	labelSelector string
}

type ConnectionUsageCalculator struct {
	headroom     int
	client       *kubernetes.Clientset
	podsSelector ConnectionUsageCalculatorPodsSelector
}

func NewConnectionUsageCalculator(
	headroom int,
	client *kubernetes.Clientset,
	podsSelector ConnectionUsageCalculatorPodsSelector,
) *ConnectionUsageCalculator {
	return &ConnectionUsageCalculator{
		headroom:     headroom,
		client:       client,
		podsSelector: podsSelector,
	}
}

func (calc *ConnectionUsageCalculator) GetConnectionUsage() int {
	usageSum := calc.headroom
	pods := calc.getMatchingPods()

	for _, pod := range pods {
		connectionsUsageString := pod.Annotations[CONNECTION_USAGE_ANNOTATION]
		if connectionsUsageString == "" {
			continue
		}

		connectionsUsage, err := strconv.Atoi(connectionsUsageString)
		if err != nil {
			fmt.Printf("Warning: Value for annotation %s is invalid: %s\n",
				CONNECTION_USAGE_ANNOTATION,
				connectionsUsageString)
			continue
		}

		usageSum += connectionsUsage
	}

	return usageSum
}

func (calc *ConnectionUsageCalculator) getMatchingPods() []v1.Pod {
	labelSelector := calc.podsSelector.labelSelector
	listOptions := metav1.ListOptions{
		LabelSelector: labelSelector,
	}
	pods, err := calc.client.CoreV1().Pods("").List(context.TODO(), listOptions)

	if errors.IsNotFound(err) {
		fmt.Printf("Warning: No pods found for label selector: %s\n", labelSelector)
		return nil
	} else if statusError, isStatus := err.(*errors.StatusError); isStatus {
		panic(fmt.Sprintf("Error listing pods %v\n", statusError.ErrStatus.Message))
	} else if err != nil {
		panic(err.Error())
	}

	return pods.Items
}
