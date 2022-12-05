#!/usr/bin/env bash

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

SCRIPT_ROOT=$(dirname "${BASH_SOURCE[0]}")/..
CODEGEN_PKG=${CODEGEN_PKG:-$(cd "${SCRIPT_ROOT}"; ls -d -1 ./vendor/k8s.io/code-generator 2>/dev/null || echo ../code-generator)}


echo "++Debug++"
echo "++${SCRIPT_ROOT}++"
echo "++${CODEGEN_PKG}++"



# generate the code with:
# --output-base    because this script should also be able to run inside the vendor dir of
#                  k8s.io/kubernetes. The output-base is needed for the generators to output into the vendor dir
#                  instead of the $GOPATH directly. For normal projects this can be dropped.
bash "${CODEGEN_PKG}"/generate-groups.sh "all" \
  kubesphere.io/scheduling/pkg/client kubesphere.io/scheduling/api \
  scheduling:v1alpha1 \
  --output-base "${SCRIPT_ROOT}/hack/tmp" \
  --go-header-file "${SCRIPT_ROOT}"/hack/boilerplate.go.txt

# To use your own boilerplate text append:
#   --go-header-file "${SCRIPT_ROOT}"/hack/custom-boilerplate.go.txt

\cp -rf "${SCRIPT_ROOT}"/hack/tmp/kubesphere.io/scheduling/api "${SCRIPT_ROOT}"/
\cp -rf "${SCRIPT_ROOT}"/hack/tmp/kubesphere.io/scheduling/pkg/client "${SCRIPT_ROOT}"/pkg
\rm -rf "${SCRIPT_ROOT}"/hack/tmp


# generate the code with:
# --output-base    because this script should also be able to run inside the vendor dir of
#                  k8s.io/kubernetes. The output-base is needed for the generators to output into the vendor dir
#                  instead of the $GOPATH directly. For normal projects this can be dropped.
#
#ECHO "-----------------------------------"
#
#bash "${CODEGEN_PKG}"/generate-groups.sh "all" \
#  kubesphere.io/scheduling/pkg/external kubesphere.io/scheduling/external/crane \
#  analysis:v1alpha1 \
#  --output-base "${SCRIPT_ROOT}/hack/tmp" \
#  --go-header-file "${SCRIPT_ROOT}"/hack/boilerplate.go.txt \
#  -v 8
#
#\cp -rf "${SCRIPT_ROOT}"/hack/tmp/kubesphere.io/scheduling/pkg/external "${SCRIPT_ROOT}"/pkg
#\rm -rf "${SCRIPT_ROOT}"/hack/tmp