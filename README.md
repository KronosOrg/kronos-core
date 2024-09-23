[![Artifact Hub](https://img.shields.io/endpoint?url=https://artifacthub.io/badge/repository/kronos-core)](https://artifacthub.io/packages/search?repo=kronos-core)
![Docker Pulls](https://img.shields.io/docker/pulls/kronosorg/kronos-core?style=flat-square&logo=docker)
[![Go Report Card](https://goreportcard.com/badge/github.com/KronosOrg/kronos-core)](https://goreportcard.com/report/github.com/KronosOrg/kronos-core)
[![Documentation](https://img.shields.io/badge/Documentation-KronosDocs-purple)](https://kronosorg.github.io/kronos-docs/)
[![Grafana](https://img.shields.io/badge/Grafana-KronosBoard-orange)](https://grafana.com/grafana/dashboards/21068-kronosboard/)

# Kronos Kubernetes Operator

Kronos is a Kubernetes operator designed to manage resource scheduling within a Kubernetes cluster. It allows users to define custom schedules for putting resources to sleep and waking them up based on specific criteria. This can be particularly useful for optimizing resource usage and costs.

## Features

- **Custom Resource Definition (CRD) Support**: Define schedules for Kubernetes resources.
- **Flexible Scheduling**: Configure sleep and wake times, weekdays, and time zones.
- **Resource Inclusion and Exclusion**: Specify which resources should follow the schedule.
- **Holiday Management**: Automatically adjust schedules for holidays.
- **Extensible**: Supports various types of Kubernetes resources.

## How It Works

To use Kronos, define your scheduling requirements using the KronosApp CRD. The CRD allows you to specify sleep and wake times, weekdays, time zones, and the resources to be managed.

### Example CRD

```yaml
apiVersion: core.wecraft.tn/v1alpha1
kind: KronosApp
metadata:
  labels:
    name: example-schedule
spec:
  startSleep: "18:00"
  endSleep: "07:00"
  weekdays: "1-5"
  timezone: "Africa/Tunis"
  includedObjects:
    - apiVersion: "apps/v1"
      kind: "Deployment"
      namespace: "default"
```
This example schedules all deployments in the default namespace to sleep from 6 PM to 7 AM CET on weekdays.

## Installation
### Kronos-Core(Operator)
#### Using Release Files
```sh
kubectl apply -f https://github.com/KronosOrg/kronos-core/releases/download/v0.4.1/kronos-core-0.4.1.yaml
```
#### Using Helm
1- Add the Helm repository:
```sh
helm repo add kronos-core https://kronosorg.github.io/kronos-chart/
```
2- Update the Helm repositories:
```sh
helm repo update
```
3- Install the operator:
```sh
helminstall <release-name> kronos-core/kronos-core --create-namespace true --namespace <installation-namespace> --version 0.3.2 -f values.yaml
```

### KronosCLI
1- Download the CLI binary:
```sh
curl -LO https://github.com/KronosOrg/kronos-cli/releases/download/v1.0.0/kronos-cli
```
2- Make the binary executable (Linux/macOS):
```sh
chmod +x kronos-cli
```
3- Move the binary to a directory in your PATH (Linux/macOS):
```sh
mv kronos-cli /usr/local/bin/
```
4- Verify the installation:
```sh
kronos-cli version
```
## Configuration
### CRD Fields
- **startSleep:** Start time for the sleep period in 24-hour format.
- **endSleep:** End time for the sleep period in 24-hour format.
- **weekdays:** Specifies weekdays for the schedule using ISO8601 format.
- **timezone:** Timezone for the schedule in IANA Timezone Database format.
- **holidays:** Array of objects specifying holidays with fields name and date.
- **includedObjects:** Array of objects specifying included Kubernetes objects.
### Example Configurations
#### Basic Configuration
Schedule all deployments in the default namespace to sleep from 6 PM to 8 AM every day.
```yaml
apiVersion: core.wecraft.tn/v1alpha1
kind: KronosApp
metadata:
  labels:
    name: basic-schedule
spec:
  startSleep: "18:00"
  endSleep: "08:00"
  weekdays: "1-7"
  timezone: "Africa/Tunis"
  includedObjects:
    - apiVersion: "apps/v1"
      kind: "Deployment"
      namespace: "default"
```
#### Advanced Configuration
Exclude deployments with "prod" in their name.
```yaml
apiVersion: core.wecraft.tn/v1alpha1
kind: KronosApp
metadata:
  labels:
    name: exclude-prod
spec:
  startSleep: "18:00"
  endSleep: "08:00"
  weekdays: "1-7"
  timezone: "Africa/Tunis"
  includedObjects:
    - apiVersion: "apps/v1"
      kind: "Deployment"
      namespace: "default"
      excludeRef: ".*prod.*"
```
## Supported Resources
Kronos supports scheduling for a variety of Kubernetes resources. These include:
- **Deployments**
- **StatefulSets**
- **ReplicaSets**
- **CronJobs**
Each of these resources can be individually included or excluded from the schedule using specific criteria defined in the KronosApp CRD. For more details, visit the [Supported Resources](https://kronosorg.github.io/kronos-docs/docs/supported-resources) page.

## Metrics
Kronos exposes metrics to monitor the operator's performance and the status of scheduled resources. The metrics can be scraped by Prometheus and visualized using Grafana.

### Available Metrics
- **schedule_info:** Provides information about the schedules applied to resources.
- **indepth_schedule_info:** Offers detailed insights into the scheduling process and resource statuses.
### Visualization
A tailored Grafana dashboard, KronosBoard, is available to visualize controller metrics and the status of KronosApp CRDs. You can find it on [Grafana's dashboard repository](https://grafana.com/grafana/dashboards/21068-kronosboard/).

## Kronos Ecosystem
Kronos is not just an operator; it is a comprehensive suite of software components and integrations that work together to provide a solution for resource scheduling in Kubernetes environments. This suite is designed to integrate with each other, offering ease of use and a robust ecosystem that enhances the overall user experience.

### Kronos-Core
At its core, Kronos includes the operator itself, Kronos-Core, which manages the scheduling of resource wake and sleep cycles. This operator is responsible for interpreting user-defined schedules and interacting with the Kubernetes API to execute them.

### Kronos-CLI
KronosCLI gives users the upper hand for forcing schedules on resources. It provides an easy way to tackle emergency situations, allowing for immediate adjustments to resource schedules.

forceWake: Immediately wakes up resources that are scheduled to be asleep, useful for debugging or troubleshooting scenarios.
forceSleep: Forces resources to enter sleep mode, overriding any existing schedules, helpful for conserving energy or addressing security concerns.
### Kronos-WebUI
A web interface, KronosWebUI, is coming soon to help users schedule resources more intuitively.

### Kronos-Chart
KronosChart is a Helm chart that simplifies the deployment of Kronos components. It can be found on Artifact Hub.

## Contributing
We welcome contributions to improve this dashboard. If you have suggestions or enhancements, please open an issue on our GitHub repository.

## License

This project is licensed under the terms of the MIT license.
