# Migration Toolkit for Red Hat OpenShift Operator
This operator is the entrypoint for installing the Migration Toolkit for Red Hat OpenShift (crane).

## A list of our components
* crane CLI
https://github.com/konveyor/crane

* The OpenShift UI Console plugin
https://github.com/konveyor/crane-ui-plugin

* A runtime image for the crane cli
https://github.com/konveyor/crane-runner/

* Proxy that allows the plugin to communicate with remote clusters
https://github.com/konveyor/crane-reverse-proxy

* Service for managing secrets related to remote clusters
https://github.com/konveyor/crane-secret-service

## Compatibility

*Note:*
- crane is compatible with OpenShift 4.10.11/4.10.11+ versions.
- Go 1.18 version needed to build operator

## Dependencies

1. Crane installs pipeline operator as dependency in `<installation>` namespace. (**Note:** This will not impact any existing installation of the pipeline operator. Uninstalling mtRHO later also will not impact the pipeline operator.)
2. Make sure `ClusterTask`, `Pipeline` CRDs are available after the installation of Pipeline operator. It might take a minute or two before the CRDs appear.

## Default Installation

Making the crane-operator available in your cluster is as simple as creating the CatalogSource:

```
oc apply -f https://raw.githubusercontent.com/konveyor/crane-operator/main/crane-catalogsource.yaml
```
Then, using the console UI, from operator hub install the crane Operator.

*Note:* 
- For now crane and MTC are not compatible within a same namespace if installed using OLM and operator hub. 

## Custom Installation

To build images from the latest code use the below instructions. 

### Building custom images

1. Set quay org, tag and image base 

    ```shell script
    export ORG=your-quay-org
    export VERSION=99.0.0
    export IMAGE_TAG_BASE=quay.io/$ORG/crane-operator
    ```

2. Run from the root of the crane-operator repo to build container image and push it.

    ```shell script
    docker build -f Dockerfile -t quay.io/$ORG/crane-operator-container:$VERSION .
    docker push  quay.io/$ORG/crane-operator-container:$VERSION
    ```
   
3. Update `/config/manager/manager.yaml` to use your own custom image for container.

    ```shell script
    [...]
       containers:
         - command:
           - /manager
           args:
           - --leader-elect
           image: quay.io/$ORG/crane-operator-container:$VERSION
           imagePullPolicy: Always
           name: manager
    ```
    
4. Run from the root of crane-operator repo to build operator bundle.
    ```shell script
    make bundle
    make bundle-build
    make bundle-push
   ```
5. Run from the root of crane-operator repo to build operator index.
   ```
    opm index add --container-tool podman --bundles quay.io/$ORG/crane-operator-bundle:v$VERSION --tag quay.io/$ORG/crane-operator-index:v$VERSION
    podman push quay.io/$ORG/crane-operator-index:v$VERSION
    ```

*Note:* Make sure your quay repos are public

## Installing Operator

1. Create CatalogSource by running the following command. 
    
    ```shell script
    cat << EOF > catalogsource.yaml
    apiVersion: operators.coreos.com/v1alpha1
    kind: CatalogSource
    metadata:
      name: crane-operator
      namespace: openshift-marketplace
    spec:
      image: 'quay.io/$ORG/crane-operator-index:v$VERSION'
      sourceType: grpc
    EOF
    
    oc create -f catalogsource.yaml
    ```

2. Install crane operator from operator hub, choose to enable console plugin while installing operator
3. Create operatorConfig CR to initiate installation of proxy, crane-ui-plugin, and cluster tasks needed for migration
    ```shell script
    cat << EOF > openshift-migration.yaml
    apiVersion: crane.konveyor.io/v1alpha1
    kind: OperatorConfig
    metadata:
      name: openshift-migration
    spec: {}
    EOF
    
    oc create -f openshift-migration.yaml
    ```
  
## Clean up

1. Remove All operatorConfig CR
    
    ```shell script
     oc delete operatorconfigs.crane.konveyor.io --all
    ```
2. Delete subscription or uninstall form operator hub.
3. Remove CRD
    ```shell script
    oc delete crd operatorconfigs.crane.konveyor.io
    ```
