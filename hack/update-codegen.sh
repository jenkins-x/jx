#!/bin/bash

# Copyright 2017 The Kubernetes Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -o errexit
set -o nounset
set -o pipefail

# update the code-genrator submodule
echo "Installing the code-generator"
VERSION='kubernetes-1.11.3'
CODE_GENERATOR='./hack/code-generator'
rm -rf ${CODE_GENERATOR} 2>/dev/null 
git clone --branch ${VERSION} git@github.com:kubernetes/code-generator.git ${CODE_GENERATOR} 2>/dev/null 

export GO111MODULE=on 
SCRIPT_ROOT=$(dirname ${BASH_SOURCE})/..
CODEGEN_PKG=${CODEGEN_PKG:-$(cd ${SCRIPT_ROOT}; ls -d -1 ./hack/code-generator 2>/dev/null || echo ../code-generator)}
OUTDIR=$(dirname ${BASH_SOURCE})/../../../..

echo "Generating code to ${OUTDIR}"

# gene
# rate the code with:
# --output-base    because this script should also be able to run inside the vendor dir of
#                  k8s.io/kubernetes. The output-base is needed for the generators to output into the vendor dir
#                  instead of the $GOPATH directly. For normal projects this can be dropped.
${CODEGEN_PKG}/generate-groups.sh all \
  github.com/jenkins-x/jx/pkg/client github.com/jenkins-x/jx/pkg/apis \
  jenkins.io:v1 \
  --output-base "${OUTDIR}" \
  --go-header-file ${SCRIPT_ROOT}/hack/custom-boilerplate.go.txt
