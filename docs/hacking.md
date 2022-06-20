### Get/Update the images that are running with crane-operator

To get the version of the images that are running within crane-operator workloads follow the below steps:

1. Run `oc get csv -n openshift-migration-toolkit`, the output should look something like 
    ```
    NAME                                     DISPLAY                       VERSION   REPLACES                                 PHASE
    crane-operator.v99.0.0                   crane                         99.0.0                                             Succeeded
    openshift-pipelines-operator-rh.v1.7.1   Red Hat OpenShift Pipelines   1.7.1     openshift-pipelines-operator-rh.v1.7.0   Succeeded
    ```
2. Run `oc get csv -n openshift-migration-toolkit crane-operator.v99.0.0 -o yaml`, the `env` and `image` field contains information about which image is getting consumed for what workload. Add/update the following `config.env` fields to configure images. 
    ```
    env:
    - name: CRANE_RUNNER_IMAGE
      value: quay.io/konveyor/crane-runner:latest
    - name: CRANE_UI_PLUGIN_IMAGE
      value: quay.io/konveyor/crane-ui-plugin:latest
    - name: CRANE_REVERSE_PROXY_IMAGE
      value: quay.io/konveyor/crane-reverse-proxy:latest
    - name: CRANE_SECRET_SERVICE_IMAGE
      value: quay.io/konveyor/crane-secret-service:latest
    image: quay.io/konveyor/crane-operator-container:latest
    ```
 3. To update any of the running images, subscription would need to be updated. Run `oc edit sub crane-operator -n openshift-migration-toolkit` to edit the subscription.
    ```shell script
    spec:
      config:
        env:
        - name: CRANE_RUNNER_IMAGE
          value: <crane-runner-image>
        - name: CRANE_UI_PLUGIN_IMAGE
          value: <crane-ui-plugin-image>
        - name: CRANE_REVERSE_PROXY_IMAGE
          value: <crane-reverse-proxy-image>
        - name: CRANE_SECRET_SERVICE_IMAGE
          value: <crane-secret-service-image>
    ```
    **Note**: Omit any of the env variables(name/value) related to images that needs to be set as default from above. 