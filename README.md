# Migration Toolkit for Red Hat Openshift Opeartor
This operator is the entrypoint for installing the Migration Toolkit for Red Hat OpenShift (mtRHO).

## Compatibility

*Note:*
- mtRHO is compatible with OpenShift 4.10.7/4.10.7+ versions.
- Go 1.18 version needed to build operator

## Prerequisites

1. Install pipeline operator. [Here](https://docs.openshift.com/container-platform/4.10/cicd/pipelines/installing-pipelines.html) are the steps to install pipeline operator.
2. Make sure `ClusterTask`, `Pipeline` CRDs are available after the installation of Pipeline operator. It might take a minute or two before the CRDs appear.

## Default Installation

Making the mtrho-operator available in your cluster is as simple as creating the CatalogSource:

```
oc apply -f https://raw.githubusercontent.com/konveyor/mtrho-operator/main/mtrho-catalogsource.yaml
```
Then, using the console UI, from operator hub install the mtRHO Operator.

*Note:* 
- For now mtRHO and MTC are not compatible within a same namespace if installed using OLM and operator hub. 
- To interact with UI properly, before creating `operatorConfig` CR, run `oc create -f https://raw.githubusercontent.com/konveyor/crane-reverse-proxy/main/dev-route.yml`. This route is a workaround of a CORS issue we are facing, there is a patch to resolve the same upstream as well, once that gets released we would no longer need to create this route.

## Custom Installation

To build images from the latest code use the below instructions. 

### Building custom images

1. Set quay org, tag and image base 

    ```shell script
    export ORG=your-quay-org
    export VERSION=99.0.0
    export IMAGE_TAG_BASE=quay.io/$ORG/mtrho-operator
    ```

2. Run from the root of the mtrho-operator repo to build container image and push it.

    ```shell script
    docker build -f Dockerfile -t quay.io/$ORG/mtrho-operator-container:$VERSION .
    docker push  quay.io/$ORG/mtrho-operator-container:$VERSION
    ```
   
3. Update `/config/manager/manager.yaml` to use your own custom image for container.

    ```shell script
    [...]
       containers:
         - command:
           - /manager
           args:
           - --leader-elect
           image: quay.io/$ORG/mtrho-operator-container:$VERSION
           imagePullPolicy: Always
           name: manager
    ```
    
4. Run from the root of mtrho-operator repo to build operator bundle.
    ```shell script
    make bundle
    make bundle-build
    make bundle-push
   ```
5. Run from the root of mtrho-operator repo to build operator index.
   ```
    opm index add --container-tool podman --bundles quay.io/$ORG/mtrho-operator-bundle:v$VERSION --tag quay.io/$ORG/mtrho-operator-index:v$VERSION
    podman push quay.io/$ORG/mtrho-operator-index:v$VERSION
    ```

*Note:* Make sure your quay repos are public

## Installing Operator

1. Create CatalogSource by running the following command. 
    
    ```shell script
    cat << EOF > catalogsource.yaml
    apiVersion: operators.coreos.com/v1alpha1
    kind: CatalogSource
    metadata:
      name: mtrho-operator
      namespace: openshift-marketplace
    spec:
      image: 'quay.io/$ORG/mtrho-operator-index:v$VERSION'
      sourceType: grpc
    EOF
    
    oc create -f catalogsource.yaml
    ```

2. Install mtRHO operator from operator hub, choose to enable console plugin while installing operator
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
