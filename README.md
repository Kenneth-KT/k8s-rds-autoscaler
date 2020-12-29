# k8s-rds-autoscaler
make RDS Aurora Serverless cluster autoscaling react quicker by scaling it actively from K8s

## Usage
### Step 1
Make sure there is a common label selector for the group of pods you are going apply RDS autoscaler.

The label "app.kubernetes.io/instance" can be used by Helm convention.

### Step 2
Specify pod connection usage via `rds-autoscaler/connections-usage` pod annotations.

Your pod specification should look like this:
```
apiVersion: v1
kind: Pod
metadata:
  annotations:
    rds-autoscaler/connections-usage: 100
...
```
### Step 3
Get an AWS access key and make sure it is allowed to perform `rds:ModifyCurrentDBClusterCapacity` operation over your RDS cluster.
### Step 4
Deploy the autoscaler via Helm
```
BUILD_TAG=$(date +%s); \
  IMAGE_URL="<MY_DOCKER_REPO_URL>"; \
  docker build -t $IMAGE_URL:$BUILD_TAG /<PATH_TO>/k8s-rds-autoscaler && \
  docker push $IMAGE_URL:$BUILD_TAG && \
  helm upgrade -i rds-autoscaler /<PATH_TO>/k8s-rds-autoscaler/deployments/helm/rds-autoscaler -n <MY_NAMESPACE> \
    --set image.repository=$IMAGE_URL,image.tag=$BUILD_TAG \
    --set autoscaler.podsLabelSelector="<MY_EXAMPLE_LABEL_1>=<MY_EXAMPLE_LABEL_VALUE_1>,<MY_EXAMPLE_LABEL_2>=<MY_EXAMPLE_LABEL_VALUE_2>" \
    --set-string autoscaler.connectionsHeadroom="50" \
    --set autoscaler.scalingsSet='[{"scale":"2"\,"limit":157}\,{"scale":"4"\,"limit":315}\,{"scale":"8"\,"limit":631}\,{"scale":"16"\,"limit":1263}\,{"scale":"32"\,"limit":2526}\,{"scale":"64"\,"limit":4816}\,{"scale":"192"\,"limit":15160}\,{"scale":"384"\,"limit":30320}]' \
    --set-string autoscaler.operationTimeout="10" \
    --set autoscaler.dbIdentifier="<MY_RDS_INSTANCE_IDENTIFIER>" \
    --set-string autoscaler.downscaleForbiddenWindowSeconds="300" \
    --set-string autoscaler.syncIntervalSeconds="3" \
    --set autoscaler.awsAccessKeyId="<MY_AWS_ACCESS_KEY_ID>" \
    --set autoscaler.awsSecretAccessKey="<MY_AWS_SECRET_KEY>" \
    --set autoscaler.awsRegion="<AWS_REGION>"
```

### Step 4
Use `kubectl logs` to check if the autoscaler is running correctly.