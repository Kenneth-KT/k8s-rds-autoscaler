package autoscaler

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go/aws/session"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type Controller struct {
	usageCalc                *ConnectionUsageCalculator
	scalingsSet              *ScalingsSet
	rdsScaleManager          *RdsScaleManager
	downscaleForbiddenWindow time.Duration
	syncInterval             time.Duration
	lastScaleCapacity        *ScalingCapacity
	lastScaleTime            *metav1.Time
}

func newController() *Controller {
	configPodsSelector := ConnectionUsageCalculatorPodsSelector{
		labelSelector: os.Getenv("PODS_LABEL_SELECTOR"),
	}
	configHeadroom, _ := strconv.Atoi(os.Getenv("CONNECTIONS_HEADROOM"))
	configScalingsSet := ParseScalingsSet(os.Getenv("SCALINGS_SET"))
	configOperationTimeout, _ := strconv.Atoi(os.Getenv("OPERATION_TIMEOUT"))
	configDbIdentifier := os.Getenv("DB_IDENTIFIER")
	configDownscaleForbiddenWindowSeconds, _ := strconv.Atoi(os.Getenv("DOWNSCALE_FORBIDDEN_WINDOW_SECONDS"))
	configDownscaleForbiddenWindow := time.Duration(configDownscaleForbiddenWindowSeconds) * time.Second
	configSyncIntervalSeconds, _ := strconv.Atoi(os.Getenv("SYNC_INTERVAL_SECONDS"))
	configSyncInterval := time.Duration(configSyncIntervalSeconds) * time.Second

	fmt.Println("Current config:")
	fmt.Printf("headroom = %d\n", configHeadroom)
	fmt.Printf("scalingsSet = %s\n", configScalingsSet.Describe())
	fmt.Printf("operationTimeout = %d\n", configOperationTimeout)
	fmt.Printf("dbIdentifier = %s\n", configDbIdentifier)
	fmt.Printf("downscaleForbiddenWindowSeconds = %d\n", configDownscaleForbiddenWindowSeconds)
	fmt.Printf("syncIntervalSeconds = %d\n", configSyncIntervalSeconds)

	usageCalc := NewConnectionUsageCalculator(
		configHeadroom, newClient(), configPodsSelector,
	)
	// required environment variables: AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY, AWS_REGION
	rdsScaleManager := NewRdsScaleManager(session.New(), configOperationTimeout, configDbIdentifier)

	return &Controller{
		usageCalc:                usageCalc,
		scalingsSet:              configScalingsSet,
		rdsScaleManager:          rdsScaleManager,
		downscaleForbiddenWindow: configDownscaleForbiddenWindow,
		syncInterval:             configSyncInterval,
		lastScaleCapacity:        &ScalingCapacity{Scale: "Unknown", ConnectionLimit: 0},
		lastScaleTime:            nil,
	}
}

func newClient() *kubernetes.Clientset {
	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err.Error())
	}

	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}
	return client
}

func (ct *Controller) runLoop() {
	fmt.Println("RDS Autoscaler started")
	for {
		ct.reconcile()
		fmt.Printf("Sleep for %d seconds\n", int(ct.syncInterval.Seconds()))
		time.Sleep(ct.syncInterval)
	}
}

func (ct *Controller) reconcile() {
	connectionUsage := ct.usageCalc.GetConnectionUsage()
	fmt.Printf("Current connection usage: %d\n", connectionUsage)
	fitScale := ct.scalingsSet.FitScale(connectionUsage)
	if fitScale == nil {
		fmt.Printf("Cannot scale up any more, current scale: %s\n", ct.lastScaleCapacity.Scale)
		return
	}

	fmt.Printf("Fit scale: %s\n", fitScale.Scale)
	ct.applyScale(fitScale)
}

func (ct *Controller) applyScale(desiredScale *ScalingCapacity) {
	shouldScale, reason := ct.shouldApplyDesiredScaleNow(desiredScale)
	if !shouldScale {
		fmt.Printf("Will not scale, reason: %s\n", reason)
		return
	}

	fmt.Printf("Need scaling: %s\n", reason)

	error := ct.rdsScaleManager.setScale(desiredScale)
	if error != nil {
		fmt.Printf("Fail to scale RDS capacity: %s\n", error)
		return
	}

	ct.updateLastScale(desiredScale)
}

func (ct *Controller) shouldApplyDesiredScaleNow(
	desiredScale *ScalingCapacity,
) (bool, string) {
	if ct.lastScaleCapacity.Scale == desiredScale.Scale {
		return false,
			fmt.Sprintf("Desired scale %s == Current scale",
				desiredScale.Scale)
	}
	if isDownscale(ct.lastScaleCapacity, desiredScale) &&
		ct.isWithinDownscaleForbiddenWindow() {
		return false,
			fmt.Sprintf(
				"Last scaled time (%s) is still within the downscale forbidden window %d seconds",
				ct.lastScaleTime.Time.Format(time.RFC3339),
				ct.downscaleForbiddenWindow,
			)
	}

	return true,
		fmt.Sprintf("Desired scale %s != current scale %s",
			desiredScale.Scale,
			ct.lastScaleCapacity.Scale)
}

func (ct *Controller) isWithinDownscaleForbiddenWindow() bool {
	return time.Now().Before(ct.downscaleReadyTime())
}

func (ct *Controller) downscaleReadyTime() time.Time {
	if ct.lastScaleTime != nil {
		return ct.lastScaleTime.Add(ct.downscaleForbiddenWindow)
	} else {
		return time.Now()
	}
}

func (ct *Controller) updateLastScale(lastScale *ScalingCapacity) {
	ct.lastScaleCapacity = lastScale
	now := metav1.NewTime(time.Now())
	ct.lastScaleTime = &now

	fmt.Printf("New scale %s successfully applied at %s\n",
		ct.lastScaleCapacity.Scale,
		ct.lastScaleTime.Time.Format(time.RFC3339))
}

func isDownscale(oldScale *ScalingCapacity, newScale *ScalingCapacity) bool {
	return newScale.ConnectionLimit < oldScale.ConnectionLimit
}

func RunController() {
	fmt.Println("RDS Autoscaler by Kenneth-KT")
	newController().runLoop()
}
