FROM quay.io/operator-framework/upstream-opm-builder

RUN opm index add \
--mode semver \
--bundles quay.io/konveyor/crane-operator-bundle:latest \
--out-dockerfile index.Dockerfile \
--generate
