#!/bin/bash

# Copyright 2016 Attic Labs, Inc. All rights reserved.
# Licensed under the Apache License, version 2.0:
# http://www.apache.org/licenses/LICENSE-2.0

# This script runs on the Noms PR Builder (http://jenkins.noms.io/job/NomsPRBuilder).

set -e

export GOPATH=${WORKSPACE}
export PATH=${PATH}:/usr/local/go/bin:/var/lib/jenkins/node-v5.11.1-linux-x64/bin
NOMS_DIR=${WORKSPACE}/src/github.com/attic-labs/noms

go version
node --version

# go list is expensive, only do it once.
GO_LIST="$(go list ./... | grep -v /vendor/ | grep -v /samples/js/)"
go build ${GO_LIST}
go test ${GO_LIST}

pushd ${NOMS_DIR}
python tools/run-all-js-tests.py
popd

# The integration test only works after the node tests because the node tests sets up samples/js/node_modules
pushd ${NOMS_DIR}/samples/js
go test ./...
popd
