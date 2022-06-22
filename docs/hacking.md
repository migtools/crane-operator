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
4. To get commitID of the image that is being used run the following command - 
    ```shell script
    docker pull <image>
    docker inspect <image> | grep commit
    ```
    For example, 
    ```shell script
    docker pull quay.io/konveyor/crane-operator-container:latest
    latest: Pulling from konveyor/crane-operator-container
    f70d60810c69: Already exists 
    545277d80005: Already exists 
    de516cc59493: Pull complete 
    80a491f42542: Pull complete 
    acd6680814d4: Pull complete 
    Digest: sha256:53c7ed89a431e032dbeeb5069342b963a1467647753bec6f436b5ec0dbcdce7a
    Status: Downloaded newer image for quay.io/konveyor/crane-operator-container:latest
    quay.io/konveyor/crane-operator-container:latest
    ```
   And then,
   ```shell script
    docker inspect quay.io/konveyor/crane-operator-container:latest | grep commit
    "io.openshift.build.commit.author": "",
    "io.openshift.build.commit.date": "",
    "io.openshift.build.commit.id": "611ab30d96f76e6904f4f43ce7001b422749b431",
    "io.openshift.build.commit.message": "",
    "io.openshift.build.commit.ref": "main",
    "io.openshift.build.commit.url": "https://github.com/openshift/ocp-build-data/commit/f02094204c5dab97e4ccadd35d135a2ef12c341f",
    "io.openshift.build.commit.author": "",
    "io.openshift.build.commit.date": "",
    "io.openshift.build.commit.id": "611ab30d96f76e6904f4f43ce7001b422749b431",
    "io.openshift.build.commit.message": "",
    "io.openshift.build.commit.ref": "main",
    "io.openshift.build.commit.url": "https://github.com/openshift/ocp-build-data/commit/f02094204c5dab97e4ccadd35d135a2ef12c341f",
    ```
    