#!/bin/bash

# do remote push based on inputs
INPUT=$1
PUSH=${INPUT:-"0"} # default to 0/false

# get version from version file
VERSION=$(cat ./.version)
GITHASH=$(git rev-parse HEAD | head -c6)
# export output
export MAJOR_TAG="${VERSION}"
export BUILD_TAG="${VERSION}-git${GITHASH}"
export MAJOR_VER="${VERSION%%.*}"
MINOR_VER="${VERSION#*.}"
export MINOR_VER="${MINOR_VER%%.*}"

TARGET=$(pwd)/target
# remove and remake output target
rm -rf $TARGET
mkdir $TARGET

# get and build static
go get
go build -a -tags netgo -ldflags "-w -X main.Version=${VERSION} -X main.GitHash=${GITHASH} -extldflags \"-static\"" -o target/gyip

# use dockerfile to create minimal scratch container
docker build --rm -t gyip/gyip:${BUILD_TAG} .

# remote push if true
if [ $PUSH -eq 1 ]; then
	TARGET_PUSH=${TARGET_PUSH:-"docker://"}
	TARGET_REGISTRY=${TARGET_REGISTRY:-"docker.io"}
	TARGET_ORG=${TARGET_ORG:-"chrisruffalo"}
	TARGET_REPO=${TARGET_REPO:-"gyip"}
	TARGET="${TARGET_REGISTRY}/${TARGET_ORG}/${TARGET_REPO}"

	docker login --username "${DOCKERUSERNAME}" --password "${DOCKERPASSWORD}" ${TARGET_REGISTRY}
	docker tag gyip/gyip:${BUILD_TAG} ${TARGET}:${BUILD_TAG}
	docker tag gyip/gyip:${BUILD_TAG} ${TARGET}:${MAJOR_TAG}
	docker tag gyip/gyip:${BUILD_TAG} ${TARGET}:${MAJOR_VER}:${MINOR_VER}
	docker tag gyip/gyip:${BUILD_TAG} ${TARGET}:${MAJOR_VER}.X
	docker tag gyip/gyip:${BUILD_TAG} ${TARGET}:latest
	docker push ${TARGET}:${BUILD_TAG}
	docker push ${TARGET}:${MAJOR_TAG}
	docker push ${TARGET}:${MAJOR_VER}.${MINOR_VER}
	docker push ${TARGET}:${MAJOR_VER}.X
	docker push ${TARGET}:latest
fi