#!/bin/bash

set -e

GV="$1"

./hack/generate_group.sh all kubesphere.io/scheduling/pkg/client kubesphere.io/scheduling/api "${GV}" --output-base=./  -h "$PWD/hack/boilerplate.go.txt"
