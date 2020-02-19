# docker-builder
A service-builder that makes it easier to build and deploy multiple services in a monorepo

## Builder config file

```json
{
    "servicename": "servicename", // The servicename in k8s. String-value
    "cluster": "controller" | "services", // Which cluster should the service be deployed to.
    "builder": {}, // How should the service be build and deployed... Optional and defaults to the default manual builder.
    "deploy": {},
    "deploymentfile": "", // Deploymentfile for kubernetes. Optional and defaults to deployment.yaml.
}
```

Manual builder
```json
{
    "type": "manual",
    "dockerfile": "", // Dockerfile to build the project. Optional and defaults to Dockerfile
    "buildcontext": "root" | "projectdir", // In which context should the dockerfile be built. In YT3 context or in projectdir context. Optional field with default set to root.
}
```

dotnet builder
```json
{
    "type": "dotnet",
    "dotnetruntime": "runtime" | "aspnet", // What framework is used by the service. By default the builder checks the dependencies of the solution and selects the best runtime for you, but you can override the checks by setting this property.
    "projectfile": "", // Name of project file relative to this settingsfile. Optional, defaults to file named *.csproj. Required only if multiple csproj files exists inside same folder.
}
```

node builder
```json
{
    "type": "nodejs",
    "nodeversion": "{nodejs version}", // Required field. What node version should be used to build and as runtime. Eg. 12 or more specific with 12.14.1
    "buildcommand": "", // Optional field. Command to build the project. Could be as simple as "npm build".
    "runcommand": "", // Required field. Command to run the project. Could be as simple as "npm start".
}

```


## Example deployment.yaml file
Servicename, namespace and image gets replaced in the build-step, while the rest are replaced as part of the release step.
```yaml

apiVersion: extensions/v1beta1
kind: Deployment
metadata:
  name: #{servicename}#
  namespace: #{namespace}#
spec:
  replicas: #{replicas}#
  template:
    metadata:
      labels:
        app: #{servicename}#
    spec:      
      containers:
      - name: #{servicename}#
        image: #{image}#
        env:
        - name: ENVIRONMENT_VALUE_TO_SET_IN_APPLICATION
          value: "#{VARIABLE_GIVEN_TO_DEPLOYSCRIPT}#"
        - name: Storage
          value: "#{Storage}#"
        - name: ServiceBus
          value: "#{ServiceBus}#"
        - name: logLevel
          value: "Warn"
        - name: ApplicationInsights__InstrumentationKey
          value: "#{ApplicationInsightsInstrumentationKey}#"
        - name: BatchSize
          value: "#{BatchSize}#"
```